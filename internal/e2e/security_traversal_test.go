package e2e

import (
	"testing"

	"github.com/Dicklesworthstone/slb/internal/core"
	"github.com/Dicklesworthstone/slb/internal/db"
)

func TestSecurityBypass_PathTraversal(t *testing.T) {
	engine := core.NewPatternEngine()

	// Effectively "rm -rf /etc"
	// We use /tmp as start (not critical) traversing to /etc (critical)
	cwd := "/home/user"
	cmd := "rm -rf /tmp/../../etc"

	// Before fix: Resolved to "rm -rf /tmp/../../etc" (regex didn't match dotdot inside).
	// Matched generic "^rm -rf" (Dangerous).

	// After fix: Resolves to "rm -rf /etc".
	// Matches "^rm ... /etc" (Critical).

	result := engine.ClassifyCommand(cmd, cwd)

	if !result.NeedsApproval {
		t.Errorf("SECURITY BYPASS: %q classified as SAFE/NO-MATCH", cmd)
	}

	if result.Tier != db.RiskTierCritical {
		t.Errorf("SECURITY WEAKNESS: %q classified as %s, want CRITICAL (Path Traversal to /etc)", cmd, result.Tier)
	}
}
