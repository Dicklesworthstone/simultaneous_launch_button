package core

import (
	"testing"
	"time"

	"github.com/Dicklesworthstone/slb/internal/db"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from db.RequestStatus
		to   db.RequestStatus
		want bool
	}{
		{"new->pending", "", db.StatusPending, true},
		{"new->approved (invalid)", "", db.StatusApproved, false},

		{"pending->approved", db.StatusPending, db.StatusApproved, true},
		{"pending->rejected", db.StatusPending, db.StatusRejected, true},
		{"pending->cancelled", db.StatusPending, db.StatusCancelled, true},
		{"pending->timeout", db.StatusPending, db.StatusTimeout, true},
		{"pending->executing (invalid)", db.StatusPending, db.StatusExecuting, false},

		{"timeout->escalated", db.StatusTimeout, db.StatusEscalated, true},
		{"timeout->approved (invalid)", db.StatusTimeout, db.StatusApproved, false},

		{"approved->executing", db.StatusApproved, db.StatusExecuting, true},
		{"approved->cancelled", db.StatusApproved, db.StatusCancelled, true},
		{"approved->executed (invalid)", db.StatusApproved, db.StatusExecuted, false},

		{"executing->executed", db.StatusExecuting, db.StatusExecuted, true},
		{"executing->execution_failed", db.StatusExecuting, db.StatusExecutionFailed, true},
		{"executing->timed_out", db.StatusExecuting, db.StatusTimedOut, true},

		{"executed->pending (invalid)", db.StatusExecuted, db.StatusPending, false},
		{"rejected->approved (invalid)", db.StatusRejected, db.StatusApproved, false},
		{"cancelled->approved (invalid)", db.StatusCancelled, db.StatusApproved, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTransition(tt.from, tt.to); got != tt.want {
				t.Fatalf("CanTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestTransitionSetsResolvedAtForTerminalStates(t *testing.T) {
	t.Run("sets created_at for new->pending when missing", func(t *testing.T) {
		req := &db.Request{Status: ""}
		if err := Transition(req, db.StatusPending); err != nil {
			t.Fatalf("Transition() error = %v", err)
		}
		if req.Status != db.StatusPending {
			t.Fatalf("Status = %q, want %q", req.Status, db.StatusPending)
		}
		if req.CreatedAt.IsZero() {
			t.Fatalf("CreatedAt is zero, want non-zero after creation transition")
		}
	})

	t.Run("sets resolved_at for rejected", func(t *testing.T) {
		req := &db.Request{Status: db.StatusPending}
		if err := Transition(req, db.StatusRejected); err != nil {
			t.Fatalf("Transition() error = %v", err)
		}
		if req.Status != db.StatusRejected {
			t.Fatalf("Status = %q, want %q", req.Status, db.StatusRejected)
		}
		if req.ResolvedAt == nil {
			t.Fatalf("ResolvedAt is nil, want non-nil for terminal state")
		}
	})

	t.Run("does not set resolved_at for timeout", func(t *testing.T) {
		req := &db.Request{Status: db.StatusPending}
		if err := Transition(req, db.StatusTimeout); err != nil {
			t.Fatalf("Transition() error = %v", err)
		}
		if req.Status != db.StatusTimeout {
			t.Fatalf("Status = %q, want %q", req.Status, db.StatusTimeout)
		}
		if req.ResolvedAt != nil {
			t.Fatalf("ResolvedAt = %v, want nil for non-terminal state", req.ResolvedAt)
		}
	})

	t.Run("sets resolved_at for executed", func(t *testing.T) {
		req := &db.Request{Status: db.StatusExecuting}
		if err := Transition(req, db.StatusExecuted); err != nil {
			t.Fatalf("Transition() error = %v", err)
		}
		if req.Status != db.StatusExecuted {
			t.Fatalf("Status = %q, want %q", req.Status, db.StatusExecuted)
		}
		if req.ResolvedAt == nil {
			t.Fatalf("ResolvedAt is nil, want non-nil for terminal state")
		}
	})

	t.Run("sets approval_expires_at on approved transition (dangerous)", func(t *testing.T) {
		req := &db.Request{Status: db.StatusPending, RiskTier: db.RiskTierDangerous}

		before := time.Now().UTC()
		if err := Transition(req, db.StatusApproved); err != nil {
			t.Fatalf("Transition() error = %v", err)
		}
		after := time.Now().UTC()

		if req.ApprovalExpiresAt == nil {
			t.Fatalf("ApprovalExpiresAt is nil, want non-nil after approved transition")
		}

		min := before.Add(defaultApprovalTTL)
		max := after.Add(defaultApprovalTTL)
		if req.ApprovalExpiresAt.Before(min) || req.ApprovalExpiresAt.After(max) {
			t.Fatalf("ApprovalExpiresAt = %s, want between [%s, %s]", req.ApprovalExpiresAt.Format(time.RFC3339), min.Format(time.RFC3339), max.Format(time.RFC3339))
		}
	})

	t.Run("sets shorter approval_expires_at for critical", func(t *testing.T) {
		req := &db.Request{Status: db.StatusPending, RiskTier: db.RiskTierCritical}

		before := time.Now().UTC()
		if err := Transition(req, db.StatusApproved); err != nil {
			t.Fatalf("Transition() error = %v", err)
		}
		after := time.Now().UTC()

		if req.ApprovalExpiresAt == nil {
			t.Fatalf("ApprovalExpiresAt is nil, want non-nil after approved transition")
		}

		min := before.Add(defaultApprovalTTLCritical)
		max := after.Add(defaultApprovalTTLCritical)
		if req.ApprovalExpiresAt.Before(min) || req.ApprovalExpiresAt.After(max) {
			t.Fatalf("ApprovalExpiresAt = %s, want between [%s, %s]", req.ApprovalExpiresAt.Format(time.RFC3339), min.Format(time.RFC3339), max.Format(time.RFC3339))
		}
	})
}

func TestTransitionRejectsInvalidMoves(t *testing.T) {
	req := &db.Request{Status: db.StatusPending}
	if err := Transition(req, db.StatusExecuting); err == nil {
		t.Fatalf("expected error for invalid transition")
	}
	if req.Status != db.StatusPending {
		t.Fatalf("Status changed on invalid transition: %q", req.Status)
	}
	if req.ResolvedAt != nil {
		t.Fatalf("ResolvedAt changed on invalid transition: %v", req.ResolvedAt)
	}
}

func TestIsTerminal(t *testing.T) {
	tests := []struct {
		status db.RequestStatus
		want   bool
	}{
		{db.StatusPending, false},
		{db.StatusApproved, false},
		{db.StatusRejected, true},
		{db.StatusExecuting, false},
		{db.StatusExecuted, true},
		{db.StatusExecutionFailed, true},
		{db.StatusCancelled, true},
		{db.StatusTimeout, false},
		{db.StatusTimedOut, true},
		{db.StatusEscalated, false},
	}

	for _, tt := range tests {
		if got := IsTerminal(tt.status); got != tt.want {
			t.Fatalf("IsTerminal(%q) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestTransitionRejectsSameState(t *testing.T) {
	req := &db.Request{Status: db.StatusPending}
	if err := Transition(req, db.StatusPending); err == nil {
		t.Fatalf("expected error for same-state transition")
	}
}
