package cli

import (
	"strings"
	"testing"

	"github.com/Dicklesworthstone/slb/internal/db"
	"github.com/Dicklesworthstone/slb/internal/testutil"
	"github.com/spf13/cobra"
)

// newTestRollbackCmd creates a fresh rollback command for testing.
func newTestRollbackCmd(dbPath string) *cobra.Command {
	root := &cobra.Command{
		Use:           "slb",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&flagDB, "db", dbPath, "database path")
	root.PersistentFlags().StringVarP(&flagOutput, "output", "o", "text", "output format")
	root.PersistentFlags().BoolVarP(&flagJSON, "json", "j", false, "json output")
	root.PersistentFlags().StringVarP(&flagProject, "project", "C", "", "project directory")

	// Create fresh rollback command
	rbCmd := &cobra.Command{
		Use:   "rollback <request-id>",
		Short: "Rollback an executed command",
		Args:  cobra.ExactArgs(1),
		RunE:  rollbackCmd.RunE,
	}
	rbCmd.Flags().BoolVarP(&flagRollbackForce, "force", "f", false, "force rollback")

	root.AddCommand(rbCmd)

	return root
}

func resetRollbackFlags() {
	flagDB = ""
	flagOutput = "text"
	flagJSON = false
	flagProject = ""
	flagRollbackForce = false
}

func TestRollbackCommand_RequiresRequestID(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRollbackFlags()

	cmd := newTestRollbackCmd(h.DBPath)
	_, _, err := executeCommand(cmd, "rollback")

	if err == nil {
		t.Fatal("expected error when request ID is missing")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRollbackCommand_RequestNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRollbackFlags()

	cmd := newTestRollbackCmd(h.DBPath)
	_, err := executeCommandCapture(t, cmd, "rollback", "nonexistent-request-id", "-j")

	if err == nil {
		t.Fatal("expected error for nonexistent request")
	}
}

func TestRollbackCommand_CannotRollbackPending(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRollbackFlags()

	sess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(h.ProjectDir),
		testutil.WithAgent("TestAgent"),
	)
	req := testutil.MakeRequest(t, h.DB, sess,
		testutil.WithCommand("echo hello", h.ProjectDir, true),
	)
	// Request is pending by default

	cmd := newTestRollbackCmd(h.DBPath)
	_, err := executeCommandCapture(t, cmd, "rollback", req.ID, "-j")

	if err == nil {
		t.Fatal("expected error when rolling back pending request")
	}
	if !strings.Contains(err.Error(), "status is pending") && !strings.Contains(err.Error(), "must be executed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRollbackCommand_RequiresRollbackData(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRollbackFlags()

	sess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(h.ProjectDir),
		testutil.WithAgent("TestAgent"),
	)
	req := testutil.MakeRequest(t, h.DB, sess,
		testutil.WithCommand("/bin/true", h.ProjectDir, true),
	)
	// Recompute hash using core.ComputeCommandHash
	req.Command.Hash = db.ComputeCommandHash(req.Command)
	h.DB.Exec(`UPDATE requests SET command_hash = ? WHERE id = ?`, req.Command.Hash, req.ID)
	h.DB.UpdateRequestStatus(req.ID, db.StatusApproved)
	h.DB.Exec(`UPDATE requests SET status = 'executed' WHERE id = ?`, req.ID)

	cmd := newTestRollbackCmd(h.DBPath)
	_, err := executeCommandCapture(t, cmd, "rollback", req.ID, "-j")

	if err == nil {
		t.Fatal("expected error when no rollback data available")
	}
	if !strings.Contains(err.Error(), "rollback data") && !strings.Contains(err.Error(), "capture-rollback") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRollbackCommand_Help(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRollbackFlags()

	cmd := newTestRollbackCmd(h.DBPath)
	stdout, _, err := executeCommand(cmd, "rollback", "--help")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "rollback") {
		t.Error("expected help to mention 'rollback'")
	}
	if !strings.Contains(stdout, "--force") {
		t.Error("expected help to mention '--force' flag")
	}
	// Check for basic rollback command description
	if !strings.Contains(stdout, "executed") {
		t.Error("expected help to mention 'executed' command")
	}
}
