package core

import (
	"os"
	"testing"

	"github.com/Dicklesworthstone/slb/internal/db"
)

func setupRequestTestDB(t *testing.T) *db.DB {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "slb-request-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	database, err := db.OpenAndMigrate(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	return database
}

func createRequestTestSession(t *testing.T, database *db.DB, name string) *db.Session {
	t.Helper()
	session := &db.Session{
		AgentName:   name,
		Program:     "test",
		Model:       "test-model",
		ProjectPath: "/test/project",
	}
	if err := database.CreateSession(session); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	return session
}

func TestCreateRequest_MissingSessionID(t *testing.T) {
	database := setupRequestTestDB(t)
	creator := NewRequestCreator(database, nil, nil, nil)

	_, err := creator.CreateRequest(CreateRequestOptions{
		Command: "rm -rf /tmp/test",
	})

	if err != ErrSessionRequired {
		t.Errorf("expected ErrSessionRequired, got: %v", err)
	}
}

func TestCreateRequest_MissingCommand(t *testing.T) {
	database := setupRequestTestDB(t)
	session := createRequestTestSession(t, database, "agent1")
	creator := NewRequestCreator(database, nil, nil, nil)

	_, err := creator.CreateRequest(CreateRequestOptions{
		SessionID: session.ID,
	})

	if err != ErrCommandRequired {
		t.Errorf("expected ErrCommandRequired, got: %v", err)
	}
}

func TestCreateRequest_SessionNotFound(t *testing.T) {
	database := setupRequestTestDB(t)
	creator := NewRequestCreator(database, nil, nil, nil)

	_, err := creator.CreateRequest(CreateRequestOptions{
		SessionID: "nonexistent-session",
		Command:   "rm -rf /tmp/test",
	})

	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestCreateRequest_BlockedAgent(t *testing.T) {
	database := setupRequestTestDB(t)
	session := createRequestTestSession(t, database, "blocked-agent")
	config := DefaultRequestCreatorConfig()
	config.BlockedAgents = []string{"blocked-agent"}
	creator := NewRequestCreator(database, nil, nil, config)

	_, err := creator.CreateRequest(CreateRequestOptions{
		SessionID: session.ID,
		Command:   "rm -rf /tmp/test",
	})

	if err == nil {
		t.Error("expected error for blocked agent")
	}
}

func TestCreateRequest_SafeCommand_Skipped(t *testing.T) {
	database := setupRequestTestDB(t)
	session := createRequestTestSession(t, database, "agent1")
	creator := NewRequestCreator(database, nil, nil, nil)

	result, err := creator.CreateRequest(CreateRequestOptions{
		SessionID: session.ID,
		Command:   "rm test.log", // .log files are safe
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Skipped {
		t.Error("expected safe command to be skipped")
	}
	if result.Request != nil {
		t.Error("expected no request for safe command")
	}
}

func TestCreateRequest_DangerousCommand_Created(t *testing.T) {
	database := setupRequestTestDB(t)
	session := createRequestTestSession(t, database, "agent1")
	creator := NewRequestCreator(database, nil, nil, nil)

	// Use git reset --hard which is dangerous (not critical)
	result, err := creator.CreateRequest(CreateRequestOptions{
		SessionID: session.ID,
		Command:   "git reset --hard HEAD~3",
		Cwd:       "/project",
		Justification: Justification{
			Reason: "Need to reset commits",
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Skipped {
		t.Error("expected dangerous command to not be skipped")
	}
	if result.Request == nil {
		t.Error("expected request to be created")
	}
	if result.Request.RiskTier != RiskTierDangerous {
		t.Errorf("expected RiskTierDangerous, got %s", result.Request.RiskTier)
	}
}

func TestCreateRequest_CriticalCommand_RequiresDifferentModel(t *testing.T) {
	database := setupRequestTestDB(t)
	session := createRequestTestSession(t, database, "agent1")
	creator := NewRequestCreator(database, nil, nil, nil)

	result, err := creator.CreateRequest(CreateRequestOptions{
		SessionID: session.ID,
		Command:   "rm -rf /etc/test",
		Cwd:       "/",
		Justification: Justification{
			Reason: "Critical cleanup",
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Request == nil {
		t.Fatal("expected request to be created")
	}
	if result.Request.RiskTier != RiskTierCritical {
		t.Errorf("expected RiskTierCritical, got %s", result.Request.RiskTier)
	}
	if !result.Request.RequireDifferentModel {
		t.Error("expected RequireDifferentModel=true for critical tier")
	}
}

func TestApplyRedaction_APIKey(t *testing.T) {
	cmd := "curl -H 'API-KEY: secret123' https://api.example.com"
	result := ApplyRedaction(cmd, nil)

	if result == cmd {
		t.Error("expected API key to be redacted")
	}
	if !containsSubstring(result, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in result, got: %s", result)
	}
}

func TestApplyRedaction_Password(t *testing.T) {
	cmd := "mysql -u root -p password=secret123"
	result := ApplyRedaction(cmd, nil)

	if result == cmd {
		t.Error("expected password to be redacted")
	}
}

func TestApplyRedaction_ConnectionString(t *testing.T) {
	cmd := "pg_dump postgres://user:pass@localhost/db"
	result := ApplyRedaction(cmd, nil)

	if result == cmd {
		t.Error("expected connection string to be redacted")
	}
}

func TestApplyRedaction_CustomPattern(t *testing.T) {
	cmd := "my-secret-token-abc123"
	result := ApplyRedaction(cmd, []string{`my-secret-[a-z0-9]+`})

	if result == cmd {
		t.Error("expected custom pattern to be redacted")
	}
}

func TestDetectSensitiveContent(t *testing.T) {
	tests := []struct {
		cmd      string
		expected bool
	}{
		{"ls -la", false},
		{"rm -rf /tmp", false},
		{"API_KEY=secret123 ./run.sh", true},
		{"curl -H 'token: abc123'", true},
		{"postgres://user:pass@host/db", true},
	}

	for _, tc := range tests {
		got := DetectSensitiveContent(tc.cmd)
		if got != tc.expected {
			t.Errorf("DetectSensitiveContent(%q) = %v, want %v", tc.cmd, got, tc.expected)
		}
	}
}

func TestParseCommandToArgv(t *testing.T) {
	tests := []struct {
		cmd      string
		expected []string
	}{
		{"ls -la", []string{"ls", "-la"}},
		{"rm -rf ./build", []string{"rm", "-rf", "./build"}},
		{"echo 'hello world'", []string{"echo", "hello world"}},
	}

	for _, tc := range tests {
		got, err := ParseCommandToArgv(tc.cmd)
		if err != nil {
			t.Errorf("ParseCommandToArgv(%q) error: %v", tc.cmd, err)
			continue
		}
		if len(got) != len(tc.expected) {
			t.Errorf("ParseCommandToArgv(%q) = %v, want %v", tc.cmd, got, tc.expected)
			continue
		}
		for i := range got {
			if got[i] != tc.expected[i] {
				t.Errorf("ParseCommandToArgv(%q)[%d] = %q, want %q", tc.cmd, i, got[i], tc.expected[i])
			}
		}
	}
}

func TestDynamicQuorum(t *testing.T) {
	database := setupRequestTestDB(t)

	// Create multiple sessions
	createRequestTestSession(t, database, "agent1")
	createRequestTestSession(t, database, "agent2")
	createRequestTestSession(t, database, "agent3")

	config := DefaultRequestCreatorConfig()
	config.DynamicQuorumEnabled = true
	config.DynamicQuorumFloor = 1

	creator := NewRequestCreator(database, nil, nil, config)

	// With 3 active sessions, dynamic quorum should allow up to 2 approvals (3-1)
	minApprovals := creator.checkDynamicQuorum(RiskTierCritical, 2, "/test/project")
	if minApprovals != 2 {
		t.Errorf("expected minApprovals=2 with 3 sessions, got %d", minApprovals)
	}
}

func TestDynamicQuorum_BelowFloor(t *testing.T) {
	database := setupRequestTestDB(t)

	// Only 1 session
	createRequestTestSession(t, database, "agent1")

	config := DefaultRequestCreatorConfig()
	config.DynamicQuorumEnabled = true
	config.DynamicQuorumFloor = 1

	creator := NewRequestCreator(database, nil, nil, config)

	// With 1 session (0 reviewers), should use floor
	minApprovals := creator.checkDynamicQuorum(RiskTierCritical, 2, "/test/project")
	if minApprovals != 1 {
		t.Errorf("expected minApprovals=1 (floor) with 1 session, got %d", minApprovals)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || containsSubstring(s[1:], substr)))
}
