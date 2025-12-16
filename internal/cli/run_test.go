package cli

import (
	"strings"
	"testing"

	"github.com/Dicklesworthstone/slb/internal/config"
	"github.com/Dicklesworthstone/slb/internal/db"
	"github.com/Dicklesworthstone/slb/internal/testutil"
	"github.com/spf13/cobra"
)

// newTestRunCmd creates a fresh run command for testing.
func newTestRunCmd(dbPath string) *cobra.Command {
	root := &cobra.Command{
		Use:           "slb",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&flagDB, "db", dbPath, "database path")
	root.PersistentFlags().StringVarP(&flagOutput, "output", "o", "text", "output format")
	root.PersistentFlags().BoolVarP(&flagJSON, "json", "j", false, "json output")
	root.PersistentFlags().StringVarP(&flagProject, "project", "C", "", "project directory")
	root.PersistentFlags().StringVarP(&flagSessionID, "session-id", "s", "", "session ID")
	root.PersistentFlags().StringVarP(&flagConfig, "config", "c", "", "config file")

	// Create fresh run command
	rCmd := &cobra.Command{
		Use:   "run <command>",
		Short: "Run a command with approval if required",
		Args:  cobra.ExactArgs(1),
		RunE:  runCmd.RunE,
	}
	rCmd.Flags().StringVar(&flagRunReason, "reason", "", "reason for command")
	rCmd.Flags().StringVar(&flagRunExpectedEffect, "expected-effect", "", "expected effect")
	rCmd.Flags().StringVar(&flagRunGoal, "goal", "", "goal")
	rCmd.Flags().StringVar(&flagRunSafety, "safety", "", "safety argument")
	rCmd.Flags().IntVar(&flagRunTimeout, "timeout", 300, "timeout seconds")
	rCmd.Flags().BoolVar(&flagRunYield, "yield", false, "yield to background")
	rCmd.Flags().StringSliceVar(&flagRunAttachFile, "attach-file", nil, "attach file")
	rCmd.Flags().StringSliceVar(&flagRunAttachContext, "attach-context", nil, "attach context")
	rCmd.Flags().StringSliceVar(&flagRunAttachScreen, "attach-screenshot", nil, "attach screenshot")

	root.AddCommand(rCmd)

	return root
}

func resetRunFlags() {
	flagDB = ""
	flagOutput = "text"
	flagJSON = false
	flagProject = ""
	flagSessionID = ""
	flagConfig = ""
	flagRunReason = ""
	flagRunExpectedEffect = ""
	flagRunGoal = ""
	flagRunSafety = ""
	flagRunTimeout = 300
	flagRunYield = false
	flagRunAttachFile = nil
	flagRunAttachContext = nil
	flagRunAttachScreen = nil
}

func TestRunCommand_RequiresCommand(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRunFlags()

	cmd := newTestRunCmd(h.DBPath)
	_, _, err := executeCommand(cmd, "run")

	if err == nil {
		t.Fatal("expected error when command is missing")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunCommand_RequiresSessionID(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRunFlags()

	cmd := newTestRunCmd(h.DBPath)
	_, _, err := executeCommand(cmd, "run", "echo hello", "-C", h.ProjectDir)

	if err == nil {
		t.Fatal("expected error when --session-id is missing")
	}
	if !strings.Contains(err.Error(), "--session-id is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunCommand_Help(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRunFlags()

	cmd := newTestRunCmd(h.DBPath)
	stdout, _, err := executeCommand(cmd, "run", "--help")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "run") {
		t.Error("expected help to mention 'run'")
	}
	if !strings.Contains(stdout, "--reason") {
		t.Error("expected help to mention '--reason' flag")
	}
	if !strings.Contains(stdout, "--timeout") {
		t.Error("expected help to mention '--timeout' flag")
	}
	if !strings.Contains(stdout, "--yield") {
		t.Error("expected help to mention '--yield' flag")
	}
	if !strings.Contains(stdout, "attach") {
		t.Error("expected help to mention attachment flags")
	}
}

// Note: TestRunCommand_InvalidSession is skipped because the run command
// calls os.Exit on errors, which would terminate the test process.
// This behavior would need to be refactored to support proper testing.

func TestToRateLimitConfig(t *testing.T) {
	// Test the helper function that converts config to rate limit config
	// This is a unit test for the internal function
	cfg := config.DefaultConfig()
	cfg.RateLimits.MaxPendingPerSession = 5
	cfg.RateLimits.MaxRequestsPerMinute = 10
	cfg.RateLimits.RateLimitAction = "reject"

	result := toRateLimitConfig(cfg)

	if result.MaxPendingPerSession != 5 {
		t.Errorf("expected MaxPendingPerSession=5, got %d", result.MaxPendingPerSession)
	}
	if result.MaxRequestsPerMinute != 10 {
		t.Errorf("expected MaxRequestsPerMinute=10, got %d", result.MaxRequestsPerMinute)
	}
}

func TestToRequestCreatorConfig(t *testing.T) {
	// Test the helper function that converts config to request creator config
	cfg := config.DefaultConfig()
	cfg.General.RequestTimeoutSecs = 1800 // 30 minutes
	cfg.General.ApprovalTTLMins = 60
	cfg.Agents.Blocked = []string{"blocked-agent"}

	result := toRequestCreatorConfig(cfg)

	if result.RequestTimeoutMinutes != 30 {
		t.Errorf("expected RequestTimeoutMinutes=30, got %d", result.RequestTimeoutMinutes)
	}
	if result.ApprovalTTLMinutes != 60 {
		t.Errorf("expected ApprovalTTLMinutes=60, got %d", result.ApprovalTTLMinutes)
	}
	if len(result.BlockedAgents) != 1 || result.BlockedAgents[0] != "blocked-agent" {
		t.Errorf("expected BlockedAgents=['blocked-agent'], got %v", result.BlockedAgents)
	}
}

func TestToRateLimitConfig_InvalidAction(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.RateLimits.RateLimitAction = "invalid-action"

	result := toRateLimitConfig(cfg)

	// Should default to "reject" for invalid action
	if result.Action != "reject" {
		t.Errorf("expected Action=reject for invalid action, got %v", result.Action)
	}
}

func TestToRateLimitConfig_QueueAction(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.RateLimits.RateLimitAction = "queue"

	result := toRateLimitConfig(cfg)

	if result.Action != "queue" {
		t.Errorf("expected Action=queue, got %v", result.Action)
	}
}

func TestToRateLimitConfig_WarnAction(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.RateLimits.RateLimitAction = "warn"

	result := toRateLimitConfig(cfg)

	if result.Action != "warn" {
		t.Errorf("expected Action=warn, got %v", result.Action)
	}
}

func TestToRequestCreatorConfig_ZeroTimeout(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.General.RequestTimeoutSecs = 0

	result := toRequestCreatorConfig(cfg)

	// Should default to 30 for zero/negative timeout
	if result.RequestTimeoutMinutes != 30 {
		t.Errorf("expected RequestTimeoutMinutes=30 for zero timeout, got %d", result.RequestTimeoutMinutes)
	}
}

func TestToRequestCreatorConfig_NegativeTimeout(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.General.RequestTimeoutSecs = -60

	result := toRequestCreatorConfig(cfg)

	// Should default to 30 for negative timeout
	if result.RequestTimeoutMinutes != 30 {
		t.Errorf("expected RequestTimeoutMinutes=30 for negative timeout, got %d", result.RequestTimeoutMinutes)
	}
}

func TestToRequestCreatorConfig_WithIntegrations(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Integrations.AgentMailEnabled = true
	cfg.Integrations.AgentMailThread = "test-thread"

	result := toRequestCreatorConfig(cfg)

	if !result.AgentMailEnabled {
		t.Error("expected AgentMailEnabled=true")
	}
	if result.AgentMailThread != "test-thread" {
		t.Errorf("expected AgentMailThread=test-thread, got %v", result.AgentMailThread)
	}
}

// -----------------------------------------------------------------------------
// evaluateRequestForExecution Tests
// -----------------------------------------------------------------------------

func TestEvaluateRequestForExecution_Approved(t *testing.T) {
	result := evaluateRequestForExecution(db.StatusApproved)

	if !result.ShouldExecute {
		t.Error("expected ShouldExecute=true for approved status")
	}
	if result.ShouldContinuePolling {
		t.Error("expected ShouldContinuePolling=false for approved status")
	}
	if !strings.Contains(result.Reason, "approved") {
		t.Errorf("expected Reason to mention 'approved', got %q", result.Reason)
	}
}

func TestEvaluateRequestForExecution_Pending(t *testing.T) {
	result := evaluateRequestForExecution(db.StatusPending)

	if result.ShouldExecute {
		t.Error("expected ShouldExecute=false for pending status")
	}
	if !result.ShouldContinuePolling {
		t.Error("expected ShouldContinuePolling=true for pending status")
	}
	if !strings.Contains(result.Reason, "pending") {
		t.Errorf("expected Reason to mention 'pending', got %q", result.Reason)
	}
}

func TestEvaluateRequestForExecution_Rejected(t *testing.T) {
	result := evaluateRequestForExecution(db.StatusRejected)

	if result.ShouldExecute {
		t.Error("expected ShouldExecute=false for rejected status")
	}
	if result.ShouldContinuePolling {
		t.Error("expected ShouldContinuePolling=false for rejected status")
	}
	if !strings.Contains(result.Reason, "terminal") {
		t.Errorf("expected Reason to mention 'terminal', got %q", result.Reason)
	}
}

func TestEvaluateRequestForExecution_Timeout(t *testing.T) {
	result := evaluateRequestForExecution(db.StatusTimeout)

	if result.ShouldExecute {
		t.Error("expected ShouldExecute=false for timeout status")
	}
	if result.ShouldContinuePolling {
		t.Error("expected ShouldContinuePolling=false for timeout status")
	}
	if !strings.Contains(result.Reason, "terminal") {
		t.Errorf("expected Reason to mention 'terminal', got %q", result.Reason)
	}
}

func TestEvaluateRequestForExecution_Cancelled(t *testing.T) {
	result := evaluateRequestForExecution(db.StatusCancelled)

	if result.ShouldExecute {
		t.Error("expected ShouldExecute=false for cancelled status")
	}
	if result.ShouldContinuePolling {
		t.Error("expected ShouldContinuePolling=false for cancelled status")
	}
}

func TestEvaluateRequestForExecution_ExecutionFailed(t *testing.T) {
	result := evaluateRequestForExecution(db.StatusExecutionFailed)

	if result.ShouldExecute {
		t.Error("expected ShouldExecute=false for execution_failed status")
	}
	if result.ShouldContinuePolling {
		t.Error("expected ShouldContinuePolling=false for execution_failed status")
	}
}

func TestEvaluateRequestForExecution_Executed(t *testing.T) {
	result := evaluateRequestForExecution(db.StatusExecuted)

	if result.ShouldExecute {
		t.Error("expected ShouldExecute=false for executed status")
	}
	if result.ShouldContinuePolling {
		t.Error("expected ShouldContinuePolling=false for executed status")
	}
}

// TestEvaluateRequestForExecution_AllStatuses is a comprehensive table-driven test
// covering all possible status values.
func TestEvaluateRequestForExecution_AllStatuses(t *testing.T) {
	tests := []struct {
		name                  string
		status                db.RequestStatus
		expectExecute         bool
		expectContinuePolling bool
	}{
		// Happy path
		{"approved", db.StatusApproved, true, false},

		// Continue polling
		{"pending", db.StatusPending, false, true},

		// Terminal states - should stop polling
		{"rejected", db.StatusRejected, false, false},
		{"timeout", db.StatusTimeout, false, false},
		{"cancelled", db.StatusCancelled, false, false},
		{"executed", db.StatusExecuted, false, false},
		{"execution_failed", db.StatusExecutionFailed, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluateRequestForExecution(tt.status)

			if result.ShouldExecute != tt.expectExecute {
				t.Errorf("ShouldExecute: expected %v, got %v", tt.expectExecute, result.ShouldExecute)
			}
			if result.ShouldContinuePolling != tt.expectContinuePolling {
				t.Errorf("ShouldContinuePolling: expected %v, got %v", tt.expectContinuePolling, result.ShouldContinuePolling)
			}
			if result.Reason == "" {
				t.Error("Reason should not be empty")
			}
		})
	}
}

// TestExecutionDecision_StructFields verifies the struct fields exist and are accessible.
func TestExecutionDecision_StructFields(t *testing.T) {
	d := ExecutionDecision{
		ShouldExecute:         true,
		ShouldContinuePolling: false,
		Reason:                "test reason",
	}

	if !d.ShouldExecute {
		t.Error("expected ShouldExecute=true")
	}
	if d.ShouldContinuePolling {
		t.Error("expected ShouldContinuePolling=false")
	}
	if d.Reason != "test reason" {
		t.Errorf("expected Reason='test reason', got %q", d.Reason)
	}
}

// TestEvaluateRequestForExecution_ReasonContainsStatus verifies that the reason
// includes useful debugging information about the status.
func TestEvaluateRequestForExecution_ReasonContainsStatus(t *testing.T) {
	// Test that terminal status reasons include the actual status
	terminalStatuses := []db.RequestStatus{
		db.StatusRejected,
		db.StatusTimeout,
		db.StatusCancelled,
		db.StatusExecuted,
		db.StatusExecutionFailed,
	}

	for _, status := range terminalStatuses {
		t.Run(string(status), func(t *testing.T) {
			result := evaluateRequestForExecution(status)
			if !strings.Contains(result.Reason, string(status)) {
				t.Errorf("expected Reason to contain status %q, got %q", status, result.Reason)
			}
		})
	}
}
