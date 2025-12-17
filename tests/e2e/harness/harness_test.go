package harness

import (
	"testing"

	"github.com/Dicklesworthstone/slb/internal/db"
)

func TestNewE2EEnvironment(t *testing.T) {
	env := NewE2EEnvironment(t)

	// Verify directories created
	env.Step("Verifying environment structure")

	env.AssertFileExists(".slb")
	env.AssertFileExists(".slb/state.db")
	env.AssertFileExists(".slb/logs")
	env.AssertFileExists(".slb/pending")

	// Verify git initialized
	env.Step("Verifying git repository")
	env.AssertFileExists(".git")

	head := env.GitHead()
	if len(head) < 7 {
		t.Errorf("GitHead too short: %s", head)
	}
	env.Result("Git HEAD: %s", head[:7])

	env.DBState()
	env.Logger.Elapsed()
}

func TestE2EEnvironment_Sessions(t *testing.T) {
	env := NewE2EEnvironment(t)

	env.Step("Creating a test session")
	sess := env.CreateSession("TestAgent", "test-program", "test-model")

	if sess.ID == "" {
		t.Error("session ID empty")
	}
	if sess.AgentName != "TestAgent" {
		t.Errorf("agent name: got %s, want TestAgent", sess.AgentName)
	}

	env.AssertActiveSessionCount(1)
	env.AssertSessionActive(sess)

	env.DBState()
}

func TestE2EEnvironment_Requests(t *testing.T) {
	env := NewE2EEnvironment(t)

	env.Step("Creating requestor session")
	requestor := env.CreateSession("Requestor", "claude-code", "opus")

	env.Step("Submitting a request")
	req := env.SubmitRequest(requestor, "rm -rf ./build", "Clean build artifacts")

	if req.ID == "" {
		t.Error("request ID empty")
	}
	env.AssertRequestStatus(req, db.StatusPending)
	env.AssertPendingCount(1)

	env.Step("Creating reviewer session")
	reviewer := env.CreateSession("Reviewer", "codex", "gpt-4")

	env.Step("Approving the request")
	_ = env.ApproveRequest(req, reviewer)

	env.AssertReviewCount(req, 1)
	env.AssertApprovalCount(req, 1)

	env.DBState()
}

func TestE2EEnvironment_GitOperations(t *testing.T) {
	env := NewE2EEnvironment(t)

	env.Step("Creating test file")
	env.WriteTestFile("test.txt", []byte("hello world"))
	env.AssertFileExists("test.txt")

	env.Step("Committing changes")
	hash1 := env.GitCommit("Add test file")

	if len(hash1) < 7 {
		t.Errorf("commit hash too short: %s", hash1)
	}

	env.Step("Creating another file")
	env.WriteTestFile("other.txt", []byte("other content"))

	env.Step("Second commit")
	hash2 := env.GitCommit("Add other file")

	if hash1 == hash2 {
		t.Error("commits should have different hashes")
	}

	env.Logger.Elapsed()
}

func TestStepLogger(t *testing.T) {
	logger := NewStepLogger(t)

	logger.Step(1, "First step")
	logger.Result("got value %d", 42)
	logger.DBState(2, 3)
	logger.Info("information")
	logger.Expected("foo", "bar", "bar", true)
	logger.Expected("fail", "a", "b", false)
	logger.Elapsed()

	// No assertions - just verify it doesn't panic
}

func TestLogBuffer(t *testing.T) {
	buf := NewLogBuffer()

	_, _ = buf.Write([]byte("test message"))
	_, _ = buf.Write([]byte("another message"))

	if len(buf.Entries()) != 2 {
		t.Errorf("expected 2 entries, got %d", len(buf.Entries()))
	}

	if !buf.Contains("test") {
		t.Error("buffer should contain 'test'")
	}

	if buf.Contains("nonexistent") {
		t.Error("buffer should not contain 'nonexistent'")
	}

	buf.Clear()
	if len(buf.Entries()) != 0 {
		t.Error("buffer should be empty after clear")
	}
}
