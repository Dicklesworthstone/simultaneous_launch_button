// Package harness provides the E2E test environment infrastructure.
package harness

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Dicklesworthstone/slb/internal/config"
	"github.com/Dicklesworthstone/slb/internal/db"
)

// DefaultTimeout is the maximum time for any single E2E test step.
const DefaultTimeout = 5 * time.Second

// E2EEnvironment is the test environment for E2E tests.
//
// It provides an isolated project directory with:
//   - SQLite database (migrated)
//   - Git repository (initialized)
//   - Config file (with short timeouts)
//   - Step logging for debugging
type E2EEnvironment struct {
	T *testing.T

	// ProjectDir is the root of the temp project
	ProjectDir string

	// SLBDir is .slb within ProjectDir
	SLBDir string

	// DB is the test database
	DB *db.DB

	// GitDir is the git repository root
	GitDir string

	// Config holds the test configuration
	Config *config.Config

	// Logger is the step logger
	Logger *StepLogger

	// stepCount tracks step numbers
	stepCount atomic.Int32

	// startTime is when the environment was created
	startTime time.Time
}

// NewE2EEnvironment creates a new isolated test environment.
//
// The environment includes:
//   - Temp project directory
//   - .slb directory structure
//   - Migrated SQLite database
//   - Initialized git repository
//   - Test config with short timeouts
//
// All resources are cleaned up automatically via t.Cleanup.
func NewE2EEnvironment(t *testing.T) *E2EEnvironment {
	t.Helper()

	projectDir := t.TempDir()
	slbDir := filepath.Join(projectDir, ".slb")

	// Create .slb directory structure
	dirs := []string{
		slbDir,
		filepath.Join(slbDir, "logs"),
		filepath.Join(slbDir, "pending"),
		filepath.Join(slbDir, "sessions"),
		filepath.Join(slbDir, "rollback"),
		filepath.Join(slbDir, "processed"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("E2E: creating %s: %v", dir, err)
		}
	}

	// Initialize database
	dbPath := filepath.Join(slbDir, "state.db")
	database, err := db.OpenAndMigrate(dbPath)
	if err != nil {
		t.Fatalf("E2E: opening database: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	// Initialize git repository
	if err := initGitRepo(projectDir); err != nil {
		t.Fatalf("E2E: initializing git: %v", err)
	}

	// Create test config with short timeouts
	cfg := testConfig()

	// Create step logger
	logger := NewStepLogger(t)

	env := &E2EEnvironment{
		T:          t,
		ProjectDir: projectDir,
		SLBDir:     slbDir,
		DB:         database,
		GitDir:     projectDir,
		Config:     cfg,
		Logger:     logger,
		startTime:  time.Now(),
	}

	logger.Info("E2E environment created at %s", projectDir)

	return env
}

// Step logs a test step with automatic numbering.
func (env *E2EEnvironment) Step(format string, args ...any) {
	env.T.Helper()
	step := env.stepCount.Add(1)
	env.Logger.Step(int(step), format, args...)
}

// Result logs a step result.
func (env *E2EEnvironment) Result(format string, args ...any) {
	env.T.Helper()
	env.Logger.Result(format, args...)
}

// DBState logs current database state counts.
func (env *E2EEnvironment) DBState() {
	env.T.Helper()

	sessions, _ := env.DB.ListActiveSessions(env.ProjectDir)
	pending, _ := env.DB.ListPendingRequests(env.ProjectDir)

	env.Logger.DBState(len(sessions), len(pending))
}

// Elapsed returns time since environment creation.
func (env *E2EEnvironment) Elapsed() time.Duration {
	return time.Since(env.startTime)
}

// CreateSession creates a session for testing.
func (env *E2EEnvironment) CreateSession(agent, program, model string) *db.Session {
	env.T.Helper()

	sess := &db.Session{
		ID:          "sess-" + randomID(8),
		AgentName:   agent,
		Program:     program,
		Model:       model,
		ProjectPath: env.ProjectDir,
	}

	if err := env.DB.CreateSession(sess); err != nil {
		env.T.Fatalf("CreateSession: %v", err)
	}

	env.Result("Session created: %s (%s/%s)", sess.ID, program, model)
	return sess
}

// SubmitRequest creates a request for testing.
func (env *E2EEnvironment) SubmitRequest(sess *db.Session, command, reason string) *db.Request {
	env.T.Helper()

	exp := time.Now().Add(5 * time.Minute)
	req := &db.Request{
		ID:                 "req-" + randomID(8),
		ProjectPath:        env.ProjectDir,
		Command:            db.CommandSpec{Raw: command, Cwd: env.ProjectDir, Shell: true},
		RequestorSessionID: sess.ID,
		RequestorAgent:     sess.AgentName,
		RequestorModel:     sess.Model,
		Justification:      db.Justification{Reason: reason},
		Status:             db.StatusPending,
		MinApprovals:       1,
		ExpiresAt:          &exp,
	}

	// Classify the command to set tier
	req.RiskTier = classifyCommand(command)

	if err := env.DB.CreateRequest(req); err != nil {
		env.T.Fatalf("SubmitRequest: %v", err)
	}

	env.Result("Request created: %s, tier=%s", req.ID, req.RiskTier)
	return req
}

// ApproveRequest creates an approval review.
func (env *E2EEnvironment) ApproveRequest(req *db.Request, reviewer *db.Session) *db.Review {
	env.T.Helper()
	return env.createReview(req, reviewer, db.DecisionApprove)
}

// RejectRequest creates a rejection review.
func (env *E2EEnvironment) RejectRequest(req *db.Request, reviewer *db.Session, reason string) *db.Review {
	env.T.Helper()
	return env.createReview(req, reviewer, db.DecisionReject)
}

func (env *E2EEnvironment) createReview(req *db.Request, reviewer *db.Session, decision db.Decision) *db.Review {
	env.T.Helper()

	rev := &db.Review{
		ID:                "rev-" + randomID(8),
		RequestID:         req.ID,
		ReviewerSessionID: reviewer.ID,
		ReviewerAgent:     reviewer.AgentName,
		ReviewerModel:     reviewer.Model,
		Decision:          decision,
	}

	if err := env.DB.CreateReview(rev); err != nil {
		env.T.Fatalf("createReview: %v", err)
	}

	env.Result("Review created: %s, decision=%s by %s", rev.ID, decision, reviewer.AgentName)
	return rev
}

// GetRequest retrieves a request by ID.
func (env *E2EEnvironment) GetRequest(id string) *db.Request {
	env.T.Helper()

	req, err := env.DB.GetRequest(id)
	if err != nil {
		env.T.Fatalf("GetRequest(%s): %v", id, err)
	}
	return req
}

// GetRequestStatus returns the current status of a request.
func (env *E2EEnvironment) GetRequestStatus(id string) db.RequestStatus {
	env.T.Helper()
	return env.GetRequest(id).Status
}

// WriteTestFile creates a file in the project directory.
func (env *E2EEnvironment) WriteTestFile(rel string, content []byte) string {
	env.T.Helper()

	abs := filepath.Join(env.ProjectDir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		env.T.Fatalf("WriteTestFile mkdir: %v", err)
	}
	if err := os.WriteFile(abs, content, 0644); err != nil {
		env.T.Fatalf("WriteTestFile: %v", err)
	}
	return abs
}

// GitCommit creates a git commit in the test repo.
func (env *E2EEnvironment) GitCommit(msg string) string {
	env.T.Helper()

	cmd := exec.Command("git", "-C", env.ProjectDir, "add", "-A")
	if out, err := cmd.CombinedOutput(); err != nil {
		env.T.Fatalf("git add: %v: %s", err, out)
	}

	cmd = exec.Command("git", "-C", env.ProjectDir, "commit", "--allow-empty", "-m", msg)
	if out, err := cmd.CombinedOutput(); err != nil {
		env.T.Fatalf("git commit: %v: %s", err, out)
	}

	cmd = exec.Command("git", "-C", env.ProjectDir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		env.T.Fatalf("git rev-parse: %v", err)
	}

	hash := string(out[:min(len(out), 8)])
	env.Result("Git commit: %s", hash)
	return hash
}

// GitHead returns the current HEAD commit hash.
func (env *E2EEnvironment) GitHead() string {
	env.T.Helper()

	cmd := exec.Command("git", "-C", env.ProjectDir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		env.T.Fatalf("GitHead: %v", err)
	}
	return string(out[:min(len(out)-1, 40)]) // trim newline
}

// initGitRepo initializes a git repository in the directory.
func initGitRepo(dir string) error {
	// Initialize repo
	cmd := exec.Command("git", "init", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return &gitError{"init", err, string(out)}
	}

	// Configure user for commits
	cmd = exec.Command("git", "-C", dir, "config", "user.email", "test@slb.local")
	if out, err := cmd.CombinedOutput(); err != nil {
		return &gitError{"config email", err, string(out)}
	}

	cmd = exec.Command("git", "-C", dir, "config", "user.name", "SLB Test")
	if out, err := cmd.CombinedOutput(); err != nil {
		return &gitError{"config name", err, string(out)}
	}

	// Initial commit
	cmd = exec.Command("git", "-C", dir, "commit", "--allow-empty", "-m", "Initial E2E test commit")
	if out, err := cmd.CombinedOutput(); err != nil {
		return &gitError{"initial commit", err, string(out)}
	}

	return nil
}

// testConfig returns a config suitable for E2E tests.
func testConfig() *config.Config {
	cfg := config.DefaultConfig()

	// Short timeouts for tests
	cfg.General.RequestTimeoutSecs = 60       // 60s instead of 30min
	cfg.General.ApprovalTTLMins = 5           // 5min instead of 30min
	cfg.General.ApprovalTTLCriticalMins = 2

	// Minimal approvals for faster tests
	cfg.Patterns.Dangerous.MinApprovals = 1
	cfg.Patterns.Critical.MinApprovals = 2

	return &cfg
}

// classifyCommand does basic tier classification for tests.
func classifyCommand(cmd string) db.RiskTier {
	// Very basic classification for E2E tests
	// Real classification is tested in unit tests
	switch {
	case containsAny(cmd, "rm -rf", "chmod -R 777", "git reset --hard"):
		return db.RiskTierCritical
	case containsAny(cmd, "rm", "chmod", "chown", "git push"):
		return db.RiskTierDangerous
	case containsAny(cmd, "make", "go build", "npm"):
		return db.RiskTierCaution
	default:
		return db.RiskTierCaution
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// randomID generates a random hex ID for test entities.
func randomID(n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	now := time.Now().UnixNano()
	for i := range b {
		b[i] = hex[(now+int64(i))%int64(len(hex))]
	}
	return string(b)
}

type gitError struct {
	op  string
	err error
	out string
}

func (e *gitError) Error() string {
	return "git " + e.op + ": " + e.err.Error() + ": " + e.out
}
