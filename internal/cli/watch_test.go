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
