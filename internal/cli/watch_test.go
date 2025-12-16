package cli

import (
	"testing"

	"github.com/Dicklesworthstone/slb/internal/db"
)

// =============================================================================
// SAFETY-CRITICAL TESTS for shouldAutoApproveCaution
// =============================================================================
//
// These tests are MANDATORY for the security of the SLB system.
// The shouldAutoApproveCaution function guards against unauthorized command
// execution. Every decision branch MUST be tested.
//
// Test coverage requirement: 100%
// =============================================================================

// TestShouldAutoApproveCaution_HappyPath verifies that a CAUTION tier
// request in pending status is eligible for auto-approval.
func TestShouldAutoApproveCaution_HappyPath(t *testing.T) {
	decision := shouldAutoApproveCaution(db.StatusPending, db.RiskTierCaution)

	if !decision.ShouldApprove {
		t.Errorf("expected ShouldApprove=true for pending CAUTION request, got false")
	}
	if decision.Reason == "" {
		t.Error("expected non-empty reason for approval decision")
	}
}

// TestShouldAutoApproveCaution_NotPending_Approved verifies that already-approved
// requests are NOT auto-approved again.
func TestShouldAutoApproveCaution_NotPending_Approved(t *testing.T) {
	decision := shouldAutoApproveCaution(db.StatusApproved, db.RiskTierCaution)

	if decision.ShouldApprove {
		t.Errorf("expected ShouldApprove=false for approved request, got true")
	}
	if decision.Reason == "" {
		t.Error("expected non-empty reason explaining denial")
	}
}

// TestShouldAutoApproveCaution_NotPending_Rejected verifies that rejected
// requests are NOT auto-approved.
func TestShouldAutoApproveCaution_NotPending_Rejected(t *testing.T) {
	decision := shouldAutoApproveCaution(db.StatusRejected, db.RiskTierCaution)

	if decision.ShouldApprove {
		t.Errorf("expected ShouldApprove=false for rejected request, got true")
	}
}

// TestShouldAutoApproveCaution_NotPending_Executed verifies that executed
// requests are NOT auto-approved.
func TestShouldAutoApproveCaution_NotPending_Executed(t *testing.T) {
	decision := shouldAutoApproveCaution(db.StatusExecuted, db.RiskTierCaution)

	if decision.ShouldApprove {
		t.Errorf("expected ShouldApprove=false for executed request, got true")
	}
}

// TestShouldAutoApproveCaution_NotPending_Timeout verifies that timed-out
// requests are NOT auto-approved.
func TestShouldAutoApproveCaution_NotPending_Timeout(t *testing.T) {
	decision := shouldAutoApproveCaution(db.StatusTimeout, db.RiskTierCaution)

	if decision.ShouldApprove {
		t.Errorf("expected ShouldApprove=false for timed out request, got true")
	}
}

// TestShouldAutoApproveCaution_NotPending_Cancelled verifies that cancelled
// requests are NOT auto-approved.
func TestShouldAutoApproveCaution_NotPending_Cancelled(t *testing.T) {
	decision := shouldAutoApproveCaution(db.StatusCancelled, db.RiskTierCaution)

	if decision.ShouldApprove {
		t.Errorf("expected ShouldApprove=false for cancelled request, got true")
	}
}

// TestShouldAutoApproveCaution_NotPending_ExecutionFailed verifies that
// failed execution requests are NOT auto-approved.
func TestShouldAutoApproveCaution_NotPending_ExecutionFailed(t *testing.T) {
	decision := shouldAutoApproveCaution(db.StatusExecutionFailed, db.RiskTierCaution)

	if decision.ShouldApprove {
		t.Errorf("expected ShouldApprove=false for execution-failed request, got true")
	}
}

// =============================================================================
// CRITICAL SECURITY TESTS: Dangerous tiers must NEVER be auto-approved
// =============================================================================

// TestShouldAutoApproveCaution_NotCaution_Dangerous is a CRITICAL security test.
// DANGEROUS tier commands MUST NEVER be auto-approved as they can cause
// significant harm (e.g., rm -rf, DROP TABLE, etc.)
func TestShouldAutoApproveCaution_NotCaution_Dangerous(t *testing.T) {
	decision := shouldAutoApproveCaution(db.StatusPending, db.RiskTierDangerous)

	if decision.ShouldApprove {
		t.Fatalf("SECURITY VIOLATION: DANGEROUS tier request was approved for auto-approval!")
	}
	if decision.Reason == "" {
		t.Error("expected non-empty reason explaining denial for dangerous tier")
	}
}

// TestShouldAutoApproveCaution_NotCaution_Critical is a CRITICAL security test.
// CRITICAL tier commands MUST NEVER be auto-approved as they pose extreme risk
// (e.g., system destruction, data loss, security breaches).
func TestShouldAutoApproveCaution_NotCaution_Critical(t *testing.T) {
	decision := shouldAutoApproveCaution(db.StatusPending, db.RiskTierCritical)

	if decision.ShouldApprove {
		t.Fatalf("SECURITY VIOLATION: CRITICAL tier request was approved for auto-approval!")
	}
	if decision.Reason == "" {
		t.Error("expected non-empty reason explaining denial for critical tier")
	}
}

// TestShouldAutoApproveCaution_NotCaution_Safe verifies that SAFE tier
// commands are NOT handled by auto-approve (they don't need approval at all).
func TestShouldAutoApproveCaution_NotCaution_Safe(t *testing.T) {
	// Safe tier uses a different constant
	decision := shouldAutoApproveCaution(db.StatusPending, db.RiskTier("safe"))

	if decision.ShouldApprove {
		t.Errorf("expected ShouldApprove=false for SAFE tier (should bypass approval entirely)")
	}
}

// =============================================================================
// Edge case and combination tests
// =============================================================================

// TestShouldAutoApproveCaution_DangerousAndNotPending verifies that even if
// a request is both not pending AND dangerous, we correctly reject it.
func TestShouldAutoApproveCaution_DangerousAndNotPending(t *testing.T) {
	decision := shouldAutoApproveCaution(db.StatusApproved, db.RiskTierDangerous)

	if decision.ShouldApprove {
		t.Errorf("expected ShouldApprove=false for non-pending dangerous request")
	}
	// The first check (not pending) should trigger
}

// TestShouldAutoApproveCaution_CriticalAndTimeout verifies rejection for
// critical tier requests that have timed out.
func TestShouldAutoApproveCaution_CriticalAndTimeout(t *testing.T) {
	decision := shouldAutoApproveCaution(db.StatusTimeout, db.RiskTierCritical)

	if decision.ShouldApprove {
		t.Errorf("expected ShouldApprove=false for timed-out critical request")
	}
}

// TestShouldAutoApproveCaution_ReasonContainsTier verifies that the reason
// includes the actual tier when rejecting based on tier.
func TestShouldAutoApproveCaution_ReasonContainsTier(t *testing.T) {
	decision := shouldAutoApproveCaution(db.StatusPending, db.RiskTierDangerous)

	if decision.ShouldApprove {
		t.Fatal("expected rejection for dangerous tier")
	}
	if decision.Reason == "" {
		t.Error("expected non-empty reason")
	}
	// The reason should mention the tier for debugging purposes
	if !contains(decision.Reason, "dangerous") && !contains(decision.Reason, "tier") {
		t.Errorf("expected reason to mention tier, got: %s", decision.Reason)
	}
}

// TestShouldAutoApproveCaution_ReasonContainsStatus verifies that the reason
// includes the actual status when rejecting based on status.
func TestShouldAutoApproveCaution_ReasonContainsStatus(t *testing.T) {
	decision := shouldAutoApproveCaution(db.StatusRejected, db.RiskTierCaution)

	if decision.ShouldApprove {
		t.Fatal("expected rejection for non-pending status")
	}
	if decision.Reason == "" {
		t.Error("expected non-empty reason")
	}
	// The reason should mention the status for debugging purposes
	if !contains(decision.Reason, "rejected") && !contains(decision.Reason, "pending") {
		t.Errorf("expected reason to mention status, got: %s", decision.Reason)
	}
}

// TestAutoApproveDecision_StructFields verifies the AutoApproveDecision struct
// can be properly initialized and accessed.
func TestAutoApproveDecision_StructFields(t *testing.T) {
	d := AutoApproveDecision{
		ShouldApprove: true,
		Reason:        "test reason",
	}

	if !d.ShouldApprove {
		t.Error("expected ShouldApprove to be true")
	}
	if d.Reason != "test reason" {
		t.Errorf("expected Reason='test reason', got %q", d.Reason)
	}
}

// =============================================================================
// Table-driven comprehensive test
// =============================================================================

// TestShouldAutoApproveCaution_AllCombinations is a comprehensive table-driven
// test that verifies ALL status/tier combinations behave correctly.
func TestShouldAutoApproveCaution_AllCombinations(t *testing.T) {
	statuses := []db.RequestStatus{
		db.StatusPending,
		db.StatusApproved,
		db.StatusRejected,
		db.StatusExecuted,
		db.StatusExecutionFailed,
		db.StatusTimeout,
		db.StatusCancelled,
	}

	tiers := []db.RiskTier{
		db.RiskTierCaution,
		db.RiskTierDangerous,
		db.RiskTierCritical,
		db.RiskTier("safe"),
	}

	for _, status := range statuses {
		for _, tier := range tiers {
			t.Run(string(status)+"_"+string(tier), func(t *testing.T) {
				decision := shouldAutoApproveCaution(status, tier)

				// ONLY pending + caution should be approved
				expectedApprove := status == db.StatusPending && tier == db.RiskTierCaution

				if decision.ShouldApprove != expectedApprove {
					t.Errorf(
						"status=%s tier=%s: expected ShouldApprove=%v, got %v (reason: %s)",
						status, tier, expectedApprove, decision.ShouldApprove, decision.Reason,
					)
				}

				// Reason should never be empty
				if decision.Reason == "" {
					t.Errorf("status=%s tier=%s: reason should not be empty", status, tier)
				}
			})
		}
	}
}

// =============================================================================
// TESTS for evaluateRequestForPolling
// =============================================================================
//
// These tests verify the polling business logic is correct.
// Every decision branch MUST be tested.
//
// Test coverage requirement: 100%
// =============================================================================

// TestEvaluateRequestForPolling_NewRequest verifies that a request not in the
// seen map is identified as new and should emit a pending event.
func TestEvaluateRequestForPolling_NewRequest(t *testing.T) {
	seen := make(map[string]db.RequestStatus)
	result := evaluateRequestForPolling("req-123", db.StatusPending, seen)

	if result.Action != PollActionEmitNew {
		t.Errorf("expected Action=PollActionEmitNew for new request, got %v", result.Action)
	}
	if result.EventType != "request_pending" {
		t.Errorf("expected EventType='request_pending', got %q", result.EventType)
	}
	if result.Reason == "" {
		t.Error("expected non-empty reason")
	}
}

// TestEvaluateRequestForPolling_StatusUnchanged verifies that a request with
// unchanged status is skipped.
func TestEvaluateRequestForPolling_StatusUnchanged(t *testing.T) {
	seen := map[string]db.RequestStatus{
		"req-123": db.StatusPending,
	}
	result := evaluateRequestForPolling("req-123", db.StatusPending, seen)

	if result.Action != PollActionSkip {
		t.Errorf("expected Action=PollActionSkip for unchanged status, got %v", result.Action)
	}
	if result.Reason == "" {
		t.Error("expected non-empty reason")
	}
}

// TestEvaluateRequestForPolling_StatusChangedToApproved verifies that a status
// change to approved emits the correct event.
func TestEvaluateRequestForPolling_StatusChangedToApproved(t *testing.T) {
	seen := map[string]db.RequestStatus{
		"req-123": db.StatusPending,
	}
	result := evaluateRequestForPolling("req-123", db.StatusApproved, seen)

	if result.Action != PollActionEmitStatusChange {
		t.Errorf("expected Action=PollActionEmitStatusChange, got %v", result.Action)
	}
	if result.EventType != "request_approved" {
		t.Errorf("expected EventType='request_approved', got %q", result.EventType)
	}
}

// TestEvaluateRequestForPolling_StatusChangedToRejected verifies rejection events.
func TestEvaluateRequestForPolling_StatusChangedToRejected(t *testing.T) {
	seen := map[string]db.RequestStatus{
		"req-123": db.StatusPending,
	}
	result := evaluateRequestForPolling("req-123", db.StatusRejected, seen)

	if result.Action != PollActionEmitStatusChange {
		t.Errorf("expected Action=PollActionEmitStatusChange, got %v", result.Action)
	}
	if result.EventType != "request_rejected" {
		t.Errorf("expected EventType='request_rejected', got %q", result.EventType)
	}
}

// TestEvaluateRequestForPolling_StatusChangedToExecuted verifies execution events.
func TestEvaluateRequestForPolling_StatusChangedToExecuted(t *testing.T) {
	seen := map[string]db.RequestStatus{
		"req-123": db.StatusApproved,
	}
	result := evaluateRequestForPolling("req-123", db.StatusExecuted, seen)

	if result.Action != PollActionEmitStatusChange {
		t.Errorf("expected Action=PollActionEmitStatusChange, got %v", result.Action)
	}
	if result.EventType != "request_executed" {
		t.Errorf("expected EventType='request_executed', got %q", result.EventType)
	}
}

// TestEvaluateRequestForPolling_StatusChangedToExecutionFailed verifies failed execution.
func TestEvaluateRequestForPolling_StatusChangedToExecutionFailed(t *testing.T) {
	seen := map[string]db.RequestStatus{
		"req-123": db.StatusApproved,
	}
	result := evaluateRequestForPolling("req-123", db.StatusExecutionFailed, seen)

	if result.Action != PollActionEmitStatusChange {
		t.Errorf("expected Action=PollActionEmitStatusChange, got %v", result.Action)
	}
	if result.EventType != "request_executed" {
		t.Errorf("expected EventType='request_executed' for failed execution, got %q", result.EventType)
	}
}

// TestEvaluateRequestForPolling_StatusChangedToTimeout verifies timeout events.
func TestEvaluateRequestForPolling_StatusChangedToTimeout(t *testing.T) {
	seen := map[string]db.RequestStatus{
		"req-123": db.StatusPending,
	}
	result := evaluateRequestForPolling("req-123", db.StatusTimeout, seen)

	if result.Action != PollActionEmitStatusChange {
		t.Errorf("expected Action=PollActionEmitStatusChange, got %v", result.Action)
	}
	if result.EventType != "request_timeout" {
		t.Errorf("expected EventType='request_timeout', got %q", result.EventType)
	}
}

// TestEvaluateRequestForPolling_StatusChangedToCancelled verifies cancellation events.
func TestEvaluateRequestForPolling_StatusChangedToCancelled(t *testing.T) {
	seen := map[string]db.RequestStatus{
		"req-123": db.StatusPending,
	}
	result := evaluateRequestForPolling("req-123", db.StatusCancelled, seen)

	if result.Action != PollActionEmitStatusChange {
		t.Errorf("expected Action=PollActionEmitStatusChange, got %v", result.Action)
	}
	if result.EventType != "request_cancelled" {
		t.Errorf("expected EventType='request_cancelled', got %q", result.EventType)
	}
}

// TestEvaluateRequestForPolling_UnknownStatusTransition verifies unknown status is skipped.
func TestEvaluateRequestForPolling_UnknownStatusTransition(t *testing.T) {
	seen := map[string]db.RequestStatus{
		"req-123": db.StatusPending,
	}
	// Use an unknown status
	result := evaluateRequestForPolling("req-123", db.RequestStatus("unknown"), seen)

	if result.Action != PollActionSkip {
		t.Errorf("expected Action=PollActionSkip for unknown status, got %v", result.Action)
	}
	if result.Reason == "" {
		t.Error("expected non-empty reason for unknown status")
	}
}

// TestEvaluateRequestForPolling_ReasonContainsStatusInfo verifies the reason is informative.
func TestEvaluateRequestForPolling_ReasonContainsStatusInfo(t *testing.T) {
	seen := map[string]db.RequestStatus{
		"req-123": db.StatusPending,
	}
	result := evaluateRequestForPolling("req-123", db.StatusApproved, seen)

	// Reason should mention the status transition
	if !contains(result.Reason, "pending") || !contains(result.Reason, "approved") {
		t.Errorf("expected reason to mention status transition, got: %s", result.Reason)
	}
}

// =============================================================================
// TESTS for statusToEventType
// =============================================================================

// TestStatusToEventType_AllStatuses tests all status to event type mappings.
func TestStatusToEventType_AllStatuses(t *testing.T) {
	tests := []struct {
		status   db.RequestStatus
		expected string
	}{
		{db.StatusApproved, "request_approved"},
		{db.StatusRejected, "request_rejected"},
		{db.StatusExecuted, "request_executed"},
		{db.StatusExecutionFailed, "request_executed"},
		{db.StatusTimeout, "request_timeout"},
		{db.StatusCancelled, "request_cancelled"},
		{db.StatusPending, ""},              // Pending is not a status change event
		{db.RequestStatus("unknown"), ""},   // Unknown status returns empty
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := statusToEventType(tt.status)
			if result != tt.expected {
				t.Errorf("statusToEventType(%s) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

// TestPollAction_Constants verifies the PollAction constants are defined correctly.
func TestPollAction_Constants(t *testing.T) {
	// Verify the constants have distinct non-empty values
	if PollActionEmitNew == "" {
		t.Error("PollActionEmitNew should not be empty")
	}
	if PollActionEmitStatusChange == "" {
		t.Error("PollActionEmitStatusChange should not be empty")
	}
	if PollActionSkip == "" {
		t.Error("PollActionSkip should not be empty")
	}
	if PollActionEmitNew == PollActionEmitStatusChange {
		t.Error("PollActionEmitNew and PollActionEmitStatusChange should be different")
	}
	if PollActionEmitNew == PollActionSkip {
		t.Error("PollActionEmitNew and PollActionSkip should be different")
	}
}

// TestRequestPollResult_StructFields verifies the struct can be initialized.
func TestRequestPollResult_StructFields(t *testing.T) {
	r := RequestPollResult{
		Action:    PollActionEmitNew,
		EventType: "request_pending",
		Reason:    "test reason",
	}

	if r.Action != PollActionEmitNew {
		t.Error("expected Action=PollActionEmitNew")
	}
	if r.EventType != "request_pending" {
		t.Errorf("expected EventType='request_pending', got %q", r.EventType)
	}
	if r.Reason != "test reason" {
		t.Errorf("expected Reason='test reason', got %q", r.Reason)
	}
}

// =============================================================================
// Table-driven comprehensive test for evaluateRequestForPolling
// =============================================================================

// TestEvaluateRequestForPolling_AllStatusTransitions tests all possible status
// transitions to ensure correct event types are generated.
func TestEvaluateRequestForPolling_AllStatusTransitions(t *testing.T) {
	statuses := []db.RequestStatus{
		db.StatusPending,
		db.StatusApproved,
		db.StatusRejected,
		db.StatusExecuted,
		db.StatusExecutionFailed,
		db.StatusTimeout,
		db.StatusCancelled,
	}

	for _, prevStatus := range statuses {
		for _, newStatus := range statuses {
			if prevStatus == newStatus {
				continue // Skip unchanged
			}
			t.Run(string(prevStatus)+"_to_"+string(newStatus), func(t *testing.T) {
				seen := map[string]db.RequestStatus{
					"req-test": prevStatus,
				}
				result := evaluateRequestForPolling("req-test", newStatus, seen)

				expectedEventType := statusToEventType(newStatus)
				if expectedEventType == "" {
					// Unknown status should skip
					if result.Action != PollActionSkip {
						t.Errorf("expected PollActionSkip for unknown status %s, got %v", newStatus, result.Action)
					}
				} else {
					// Known status should emit event
					if result.Action != PollActionEmitStatusChange {
						t.Errorf("expected PollActionEmitStatusChange for %s->%s, got %v", prevStatus, newStatus, result.Action)
					}
					if result.EventType != expectedEventType {
						t.Errorf("expected EventType=%q, got %q", expectedEventType, result.EventType)
					}
				}

				// Reason should never be empty
				if result.Reason == "" {
					t.Error("reason should not be empty")
				}
			})
		}
	}
}

// contains is a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
