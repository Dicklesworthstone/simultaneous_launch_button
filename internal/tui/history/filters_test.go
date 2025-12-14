package history

import (
	"strings"
	"testing"

	"github.com/Dicklesworthstone/slb/internal/db"
)

func TestNewFilters(t *testing.T) {
	f := NewFilters()

	if f.TierFilter != "" {
		t.Error("TierFilter should be empty by default")
	}
	if f.StatusFilter != "" {
		t.Error("StatusFilter should be empty by default")
	}
	if f.tierIdx != 0 {
		t.Error("tierIdx should be 0 by default")
	}
	if f.statusIdx != 0 {
		t.Error("statusIdx should be 0 by default")
	}
}

func TestCycleTier(t *testing.T) {
	f := NewFilters()

	// Cycle through all tier options
	for i := 0; i < len(TierOptions); i++ {
		expectedTier := TierOptions[(i+1)%len(TierOptions)]
		f.CycleTier()
		if f.TierFilter != expectedTier {
			t.Errorf("After cycle %d, expected tier %q, got %q", i+1, expectedTier, f.TierFilter)
		}
	}

	// After cycling through all, should be back to empty
	if f.TierFilter != "" {
		t.Errorf("After full cycle, expected empty tier, got %q", f.TierFilter)
	}
}

func TestCycleStatus(t *testing.T) {
	f := NewFilters()

	// Cycle through all status options
	for i := 0; i < len(StatusOptions); i++ {
		expectedStatus := StatusOptions[(i+1)%len(StatusOptions)]
		f.CycleStatus()
		if f.StatusFilter != expectedStatus {
			t.Errorf("After cycle %d, expected status %q, got %q", i+1, expectedStatus, f.StatusFilter)
		}
	}

	// After cycling through all, should be back to empty
	if f.StatusFilter != "" {
		t.Errorf("After full cycle, expected empty status, got %q", f.StatusFilter)
	}
}

func TestSetTier(t *testing.T) {
	f := NewFilters()

	// Set to valid tier
	f.SetTier(string(db.RiskTierCritical))
	if f.TierFilter != string(db.RiskTierCritical) {
		t.Errorf("expected tier %q, got %q", db.RiskTierCritical, f.TierFilter)
	}

	// Set to empty
	f.SetTier("")
	if f.TierFilter != "" {
		t.Errorf("expected empty tier, got %q", f.TierFilter)
	}

	// Set to unknown tier - should still set it but idx stays 0
	f.SetTier("unknown")
	if f.TierFilter != "unknown" {
		t.Errorf("expected tier 'unknown', got %q", f.TierFilter)
	}
	if f.tierIdx != 0 {
		t.Errorf("tierIdx should be 0 for unknown tier, got %d", f.tierIdx)
	}
}

func TestSetStatus(t *testing.T) {
	f := NewFilters()

	// Set to valid status
	f.SetStatus(string(db.StatusPending))
	if f.StatusFilter != string(db.StatusPending) {
		t.Errorf("expected status %q, got %q", db.StatusPending, f.StatusFilter)
	}

	// Set to empty
	f.SetStatus("")
	if f.StatusFilter != "" {
		t.Errorf("expected empty status, got %q", f.StatusFilter)
	}

	// Set to unknown status - should still set it but idx stays 0
	f.SetStatus("unknown")
	if f.StatusFilter != "unknown" {
		t.Errorf("expected status 'unknown', got %q", f.StatusFilter)
	}
	if f.statusIdx != 0 {
		t.Errorf("statusIdx should be 0 for unknown status, got %d", f.statusIdx)
	}
}

func TestClear(t *testing.T) {
	f := NewFilters()
	f.SetTier(string(db.RiskTierCritical))
	f.SetStatus(string(db.StatusPending))

	f.Clear()

	if f.TierFilter != "" {
		t.Errorf("TierFilter should be empty after Clear, got %q", f.TierFilter)
	}
	if f.StatusFilter != "" {
		t.Errorf("StatusFilter should be empty after Clear, got %q", f.StatusFilter)
	}
	if f.tierIdx != 0 {
		t.Errorf("tierIdx should be 0 after Clear, got %d", f.tierIdx)
	}
	if f.statusIdx != 0 {
		t.Errorf("statusIdx should be 0 after Clear, got %d", f.statusIdx)
	}
}

func TestHasFilters(t *testing.T) {
	f := NewFilters()

	if f.HasFilters() {
		t.Error("HasFilters should be false with no filters")
	}

	f.SetTier(string(db.RiskTierCritical))
	if !f.HasFilters() {
		t.Error("HasFilters should be true with tier filter")
	}

	f.Clear()
	f.SetStatus(string(db.StatusPending))
	if !f.HasFilters() {
		t.Error("HasFilters should be true with status filter")
	}

	f.Clear()
	if f.HasFilters() {
		t.Error("HasFilters should be false after Clear")
	}
}

func TestRenderTierBadge(t *testing.T) {
	tests := []struct {
		tier     string
		expected string
	}{
		{"", "All Tiers"},
		{string(db.RiskTierCritical), string(db.RiskTierCritical)},
		{string(db.RiskTierDangerous), string(db.RiskTierDangerous)},
		{string(db.RiskTierCaution), string(db.RiskTierCaution)},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			f := NewFilters()
			if tc.tier != "" {
				f.SetTier(tc.tier)
			}

			result := f.RenderTierBadge()
			if result == "" {
				t.Error("RenderTierBadge returned empty string")
			}
			if !strings.Contains(result, tc.expected) {
				t.Errorf("RenderTierBadge should contain %q", tc.expected)
			}
		})
	}
}

func TestRenderStatusBadge(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"", "All Status"},
		{string(db.StatusPending), "Pending"},
		{string(db.StatusApproved), "Approved"},
		{string(db.StatusRejected), "Rejected"},
		{string(db.StatusExecuted), "Executed"},
		{string(db.StatusExecutionFailed), "Failed"},
		{string(db.StatusTimeout), "Timeout"},
		{string(db.StatusEscalated), "Escalated"},
		{string(db.StatusCancelled), "Cancelled"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			f := NewFilters()
			if tc.status != "" {
				f.SetStatus(tc.status)
			}

			result := f.RenderStatusBadge()
			if result == "" {
				t.Error("RenderStatusBadge returned empty string")
			}
			if !strings.Contains(result, tc.expected) {
				t.Errorf("RenderStatusBadge should contain %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		status   db.RequestStatus
		expected string
	}{
		{db.StatusPending, "Pending"},
		{db.StatusApproved, "Approved"},
		{db.StatusRejected, "Rejected"},
		{db.StatusExecuted, "Executed"},
		{db.StatusExecuting, "Executing"},
		{db.StatusExecutionFailed, "Failed"},
		{db.StatusTimeout, "Timeout"},
		{db.StatusEscalated, "Escalated"},
		{db.StatusCancelled, "Cancelled"},
		{db.StatusTimedOut, "Timed Out"},
		{"unknown", "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			got := statusLabel(tc.status)
			if got != tc.expected {
				t.Errorf("statusLabel(%q): expected %q, got %q", tc.status, tc.expected, got)
			}
		})
	}
}

func TestTierOptionsContents(t *testing.T) {
	// Verify TierOptions has expected tiers
	if len(TierOptions) < 4 {
		t.Errorf("TierOptions should have at least 4 options, got %d", len(TierOptions))
	}

	// First should be empty (all)
	if TierOptions[0] != "" {
		t.Errorf("First TierOption should be empty, got %q", TierOptions[0])
	}

	// Should contain CRITICAL
	found := false
	for _, opt := range TierOptions {
		if opt == string(db.RiskTierCritical) {
			found = true
			break
		}
	}
	if !found {
		t.Error("TierOptions should contain CRITICAL")
	}
}

func TestStatusOptionsContents(t *testing.T) {
	// Verify StatusOptions has expected statuses
	if len(StatusOptions) < 5 {
		t.Errorf("StatusOptions should have at least 5 options, got %d", len(StatusOptions))
	}

	// First should be empty (all)
	if StatusOptions[0] != "" {
		t.Errorf("First StatusOption should be empty, got %q", StatusOptions[0])
	}

	// Should contain PENDING
	found := false
	for _, opt := range StatusOptions {
		if opt == string(db.StatusPending) {
			found = true
			break
		}
	}
	if !found {
		t.Error("StatusOptions should contain PENDING")
	}
}
