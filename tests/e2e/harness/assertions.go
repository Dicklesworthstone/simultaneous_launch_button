package harness

import (
	"os"
	"path/filepath"

	"github.com/Dicklesworthstone/slb/internal/db"
)

// AssertRequestTier verifies the request has the expected risk tier.
func (env *E2EEnvironment) AssertRequestTier(req *db.Request, expected db.RiskTier) {
	env.T.Helper()

	ok := req.RiskTier == expected
	env.Logger.Expected("tier", expected, req.RiskTier, ok)

	if !ok {
		env.T.Errorf("expected tier %s, got %s", expected, req.RiskTier)
	}
}

// AssertRequestStatus verifies the request has the expected status.
func (env *E2EEnvironment) AssertRequestStatus(req *db.Request, expected db.RequestStatus) {
	env.T.Helper()

	// Refresh request from DB
	current := env.GetRequest(req.ID)
	ok := current.Status == expected
	env.Logger.Expected("status", expected, current.Status, ok)

	if !ok {
		env.T.Errorf("expected status %s, got %s", expected, current.Status)
	}
}

// AssertRequestStatusByID verifies request status by ID.
func (env *E2EEnvironment) AssertRequestStatusByID(id string, expected db.RequestStatus) {
	env.T.Helper()

	current := env.GetRequest(id)
	ok := current.Status == expected
	env.Logger.Expected("status", expected, current.Status, ok)

	if !ok {
		env.T.Errorf("request %s: expected status %s, got %s", id, expected, current.Status)
	}
}

// AssertSessionActive verifies a session is active (not ended).
func (env *E2EEnvironment) AssertSessionActive(sess *db.Session) {
	env.T.Helper()

	current, err := env.DB.GetSession(sess.ID)
	if err != nil {
		env.T.Fatalf("AssertSessionActive: %v", err)
	}

	active := current.EndedAt == nil
	env.Logger.Expected("active", true, active, active)

	if !active {
		env.T.Errorf("session %s is not active (ended at %v)", sess.ID, current.EndedAt)
	}
}

// AssertSessionEnded verifies a session has ended.
func (env *E2EEnvironment) AssertSessionEnded(sess *db.Session) {
	env.T.Helper()

	current, err := env.DB.GetSession(sess.ID)
	if err != nil {
		env.T.Fatalf("AssertSessionEnded: %v", err)
	}

	ended := current.EndedAt != nil
	env.Logger.Expected("ended", true, ended, ended)

	if !ended {
		env.T.Errorf("session %s has not ended", sess.ID)
	}
}

// AssertReviewCount verifies the number of reviews for a request.
func (env *E2EEnvironment) AssertReviewCount(req *db.Request, expected int) {
	env.T.Helper()

	reviews, err := env.DB.ListReviewsForRequest(req.ID)
	if err != nil {
		env.T.Fatalf("AssertReviewCount: %v", err)
	}

	ok := len(reviews) == expected
	env.Logger.Expected("review count", expected, len(reviews), ok)

	if !ok {
		env.T.Errorf("expected %d reviews, got %d", expected, len(reviews))
	}
}

// AssertApprovalCount verifies the number of approvals for a request.
func (env *E2EEnvironment) AssertApprovalCount(req *db.Request, expected int) {
	env.T.Helper()

	approvals, _, err := env.DB.CountReviewsByDecision(req.ID)
	if err != nil {
		env.T.Fatalf("AssertApprovalCount: %v", err)
	}

	ok := approvals == expected
	env.Logger.Expected("approval count", expected, approvals, ok)

	if !ok {
		env.T.Errorf("expected %d approvals, got %d", expected, approvals)
	}
}

// AssertPendingCount verifies the number of pending requests.
func (env *E2EEnvironment) AssertPendingCount(expected int) {
	env.T.Helper()

	pending, err := env.DB.ListPendingRequests(env.ProjectDir)
	if err != nil {
		env.T.Fatalf("AssertPendingCount: %v", err)
	}

	ok := len(pending) == expected
	env.Logger.Expected("pending count", expected, len(pending), ok)

	if !ok {
		env.T.Errorf("expected %d pending requests, got %d", expected, len(pending))
	}
}

// AssertActiveSessionCount verifies the number of active sessions.
func (env *E2EEnvironment) AssertActiveSessionCount(expected int) {
	env.T.Helper()

	sessions, err := env.DB.ListActiveSessions(env.ProjectDir)
	if err != nil {
		env.T.Fatalf("AssertActiveSessionCount: %v", err)
	}

	ok := len(sessions) == expected
	env.Logger.Expected("active sessions", expected, len(sessions), ok)

	if !ok {
		env.T.Errorf("expected %d active sessions, got %d", expected, len(sessions))
	}
}

// AssertGitHead verifies the current git HEAD.
func (env *E2EEnvironment) AssertGitHead(expected string) {
	env.T.Helper()

	head := env.GitHead()
	// Compare first 7 chars like git shorthand
	cmpLen := min(len(expected), len(head), 7)
	ok := head[:cmpLen] == expected[:cmpLen]
	env.Logger.Expected("git HEAD", expected[:cmpLen], head[:cmpLen], ok)

	if !ok {
		env.T.Errorf("expected HEAD %s, got %s", expected[:cmpLen], head[:cmpLen])
	}
}

// AssertFileExists verifies a file exists relative to project dir.
func (env *E2EEnvironment) AssertFileExists(rel string) {
	env.T.Helper()

	path := env.MustPath(rel)
	_, err := os.Stat(path)
	ok := err == nil
	env.Logger.Expected("file exists", rel, ok, ok)

	if !ok {
		env.T.Errorf("file %s does not exist: %v", rel, err)
	}
}

// AssertFileNotExists verifies a file does not exist.
func (env *E2EEnvironment) AssertFileNotExists(rel string) {
	env.T.Helper()

	path := env.MustPath(rel)
	_, err := os.Stat(path)
	ok := os.IsNotExist(err)
	env.Logger.Expected("file not exists", rel, ok, ok)

	if !ok {
		env.T.Errorf("file %s exists but should not", rel)
	}
}

// MustPath returns the absolute path relative to project dir.
func (env *E2EEnvironment) MustPath(rel string) string {
	env.T.Helper()
	return filepath.Join(env.ProjectDir, rel)
}

// AssertNoError fails if err is non-nil.
func (env *E2EEnvironment) AssertNoError(err error, msg string) {
	env.T.Helper()
	if err != nil {
		env.Logger.Error("%s: %v", msg, err)
		env.T.Fatalf("%s: %v", msg, err)
	}
}

// AssertError fails if err is nil.
func (env *E2EEnvironment) AssertError(err error, msg string) {
	env.T.Helper()
	if err == nil {
		env.Logger.Error("%s: expected error but got nil", msg)
		env.T.Fatalf("%s: expected error but got nil", msg)
	}
}
