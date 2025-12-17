// Package e2e contains end-to-end integration tests for SLB workflows.
package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Dicklesworthstone/slb/internal/core"
	"github.com/Dicklesworthstone/slb/internal/db"
	"github.com/Dicklesworthstone/slb/internal/testutil"
)

// TestGitRollback_PointCreation tests that rollback points are created
// before executing git-related commands.
func TestGitRollback_PointCreation(t *testing.T) {
	h := testutil.NewHarness(t)

	t.Log("=== TestGitRollback_PointCreation ===")
	t.Logf("ENV: temp_db=%s", h.DBPath)

	// Step 1: Initialize test git repo
	t.Log("STEP 1: Initializing test git repo")
	repoDir := setupTestGitRepo(t, h.ProjectDir)
	t.Logf("  Git repo created: %s", repoDir)

	// Create initial files
	testFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	gitAdd(t, repoDir, "test.txt")
	gitCommit(t, repoDir, "Initial commit")
	t.Log("  Initial files committed")

	// Step 2: Create session and request for git command
	t.Log("STEP 2: Creating session and request")
	sess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(repoDir),
		testutil.WithAgent("git-test-agent"),
		testutil.WithModel("test-model"),
	)

	req := testutil.MakeRequest(t, h.DB, sess,
		testutil.WithCommand("git reset --hard HEAD~1", repoDir, true),
		testutil.WithRisk(db.RiskTierDangerous),
		testutil.WithMinApprovals(1),
	)
	t.Logf("  Request created: %s", req.ID)

	// Step 3: Capture rollback state
	t.Log("STEP 3: Capturing rollback state")
	rollbackData, err := core.CaptureRollbackState(context.Background(), req, core.RollbackCaptureOptions{})
	if err != nil {
		t.Fatalf("CaptureRollbackState failed: %v", err)
	}
	if rollbackData == nil {
		t.Fatal("Expected rollback data to be captured for git command")
	}
	t.Logf("  Rollback captured: %s", rollbackData.RollbackPath)
	t.Logf("  Kind: %s", rollbackData.Kind)

	// Step 4: Verify rollback metadata
	t.Log("STEP 4: Verifying rollback metadata")
	if rollbackData.Kind != "git" {
		t.Errorf("Expected kind=git, got %s", rollbackData.Kind)
	}
	if rollbackData.Git == nil {
		t.Fatal("Expected git rollback data to be present")
	}
	if rollbackData.Git.Head == "" {
		t.Error("Expected HEAD to be captured")
	}
	t.Logf("  HEAD: %s", rollbackData.Git.Head)
	t.Logf("  Branch: %s", rollbackData.Git.Branch)

	// Step 5: Verify rollback files exist
	t.Log("STEP 5: Verifying rollback files")
	metadataPath := filepath.Join(rollbackData.RollbackPath, "metadata.json")
	if _, err := os.Stat(metadataPath); err != nil {
		t.Errorf("Rollback metadata file not found: %v", err)
	}
	t.Log("  metadata.json exists")

	gitDir := filepath.Join(rollbackData.RollbackPath, "git")
	headFile := filepath.Join(gitDir, "head.txt")
	if _, err := os.Stat(headFile); err != nil {
		t.Errorf("Git head file not found: %v", err)
	}
	t.Log("  git/head.txt exists")

	t.Log("=== PASS: TestGitRollback_PointCreation ===")
}

// TestGitRollback_SuccessfulRestore tests that git state can be restored
// after executing a destructive command.
func TestGitRollback_SuccessfulRestore(t *testing.T) {
	h := testutil.NewHarness(t)

	t.Log("=== TestGitRollback_SuccessfulRestore ===")
	t.Logf("ENV: temp_db=%s", h.DBPath)

	// Step 1: Initialize test git repo with history
	t.Log("STEP 1: Initializing test git repo with history")
	repoDir := setupTestGitRepo(t, h.ProjectDir)

	// Create and commit file1
	file1 := filepath.Join(repoDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("hello\n"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	gitAdd(t, repoDir, "file1.txt")
	gitCommit(t, repoDir, "Add file1")
	t.Log("  file1.txt committed")

	// Create and commit file2
	file2 := filepath.Join(repoDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("world\n"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}
	gitAdd(t, repoDir, "file2.txt")
	gitCommit(t, repoDir, "Add file2")
	t.Log("  file2.txt committed")

	// Get current HEAD
	initialHead := gitHead(t, repoDir)
	t.Logf("  Initial HEAD: %s", initialHead[:8])

	// Step 2: Create request for git reset
	t.Log("STEP 2: Creating request for git reset")
	sess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(repoDir),
		testutil.WithAgent("restore-agent"),
		testutil.WithModel("test-model"),
	)

	req := testutil.MakeRequest(t, h.DB, sess,
		testutil.WithCommand("git reset --hard HEAD~1", repoDir, true),
		testutil.WithRisk(db.RiskTierDangerous),
		testutil.WithMinApprovals(1),
	)

	// Step 3: Capture rollback state before execution
	t.Log("STEP 3: Capturing rollback state")
	rollbackData, err := core.CaptureRollbackState(context.Background(), req, core.RollbackCaptureOptions{})
	if err != nil {
		t.Fatalf("CaptureRollbackState failed: %v", err)
	}
	t.Logf("  Rollback HEAD: %s", rollbackData.Git.Head[:8])

	// Step 4: Execute the destructive command (simulate)
	t.Log("STEP 4: Executing destructive git reset")
	cmd := exec.Command("git", "-C", repoDir, "reset", "--hard", "HEAD~1")
	if err := cmd.Run(); err != nil {
		t.Fatalf("git reset failed: %v", err)
	}

	// Verify file2 is gone
	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Error("file2.txt should be deleted after git reset")
	}
	t.Log("  file2.txt deleted by git reset")

	newHead := gitHead(t, repoDir)
	t.Logf("  New HEAD: %s", newHead[:8])

	// Step 5: Restore from rollback
	t.Log("STEP 5: Restoring from rollback")
	err = core.RestoreRollbackState(context.Background(), rollbackData, core.RollbackRestoreOptions{
		Force: true,
	})
	if err != nil {
		t.Fatalf("RestoreRollbackState failed: %v", err)
	}
	t.Log("  Rollback restored")

	// Step 6: Verify state restored
	t.Log("STEP 6: Verifying state restored")
	restoredHead := gitHead(t, repoDir)
	if restoredHead != initialHead {
		t.Errorf("HEAD not restored: got %s, want %s", restoredHead[:8], initialHead[:8])
	}
	t.Logf("  Restored HEAD: %s", restoredHead[:8])

	// Verify file2 is back
	if _, err := os.Stat(file2); err != nil {
		t.Errorf("file2.txt should be restored: %v", err)
	}
	content, _ := os.ReadFile(file2)
	if strings.TrimSpace(string(content)) != "world" {
		t.Errorf("file2.txt content wrong: got %q, want %q", string(content), "world")
	}
	t.Log("  file2.txt restored with correct content")

	t.Log("=== PASS: TestGitRollback_SuccessfulRestore ===")
}

// TestFilesystemRollback_PointCreation tests that filesystem rollback points
// are created for rm commands.
func TestFilesystemRollback_PointCreation(t *testing.T) {
	h := testutil.NewHarness(t)

	t.Log("=== TestFilesystemRollback_PointCreation ===")
	t.Logf("ENV: temp_db=%s", h.DBPath)

	// Step 1: Create test files
	t.Log("STEP 1: Creating test files")
	testDir := filepath.Join(h.ProjectDir, "testdata")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create testdata dir: %v", err)
	}

	file1 := filepath.Join(testDir, "important.txt")
	if err := os.WriteFile(file1, []byte("important data\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	t.Logf("  Created: %s", file1)

	// Step 2: Create session and request
	t.Log("STEP 2: Creating session and request for rm command")
	sess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(h.ProjectDir),
		testutil.WithAgent("rm-test-agent"),
		testutil.WithModel("test-model"),
	)

	req := testutil.MakeRequest(t, h.DB, sess,
		testutil.WithCommand("rm -rf "+testDir, h.ProjectDir, true),
		testutil.WithRisk(db.RiskTierDangerous),
		testutil.WithMinApprovals(1),
	)
	t.Logf("  Request created: %s", req.ID)

	// Step 3: Capture rollback state
	t.Log("STEP 3: Capturing rollback state")
	rollbackData, err := core.CaptureRollbackState(context.Background(), req, core.RollbackCaptureOptions{})
	if err != nil {
		t.Fatalf("CaptureRollbackState failed: %v", err)
	}
	if rollbackData == nil {
		t.Fatal("Expected rollback data to be captured for rm command")
	}
	t.Logf("  Rollback captured: %s", rollbackData.RollbackPath)
	t.Logf("  Kind: %s", rollbackData.Kind)

	// Step 4: Verify filesystem rollback data
	t.Log("STEP 4: Verifying filesystem rollback data")
	if rollbackData.Kind != "filesystem" {
		t.Errorf("Expected kind=filesystem, got %s", rollbackData.Kind)
	}
	if rollbackData.Filesystem == nil {
		t.Fatal("Expected filesystem rollback data to be present")
	}
	if rollbackData.Filesystem.TotalBytes == 0 {
		t.Error("Expected TotalBytes > 0")
	}
	t.Logf("  TotalBytes: %d", rollbackData.Filesystem.TotalBytes)
	t.Logf("  Roots: %d", len(rollbackData.Filesystem.Roots))

	// Step 5: Verify tar.gz exists
	t.Log("STEP 5: Verifying tar.gz backup")
	tarPath := filepath.Join(rollbackData.RollbackPath, rollbackData.Filesystem.TarGz)
	info, err := os.Stat(tarPath)
	if err != nil {
		t.Errorf("Tar.gz file not found: %v", err)
	} else {
		t.Logf("  files.tar.gz size: %d bytes", info.Size())
	}

	t.Log("=== PASS: TestFilesystemRollback_PointCreation ===")
}

// TestFilesystemRollback_SuccessfulRestore tests that files can be restored
// after deletion.
func TestFilesystemRollback_SuccessfulRestore(t *testing.T) {
	h := testutil.NewHarness(t)

	t.Log("=== TestFilesystemRollback_SuccessfulRestore ===")
	t.Logf("ENV: temp_db=%s", h.DBPath)

	// Step 1: Create test files
	t.Log("STEP 1: Creating test files")
	testDir := filepath.Join(h.ProjectDir, "restore_test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	file1 := filepath.Join(testDir, "file1.txt")
	file2 := filepath.Join(testDir, "file2.txt")
	if err := os.WriteFile(file1, []byte("content1\n"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2\n"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}
	t.Log("  Created file1.txt and file2.txt")

	// Step 2: Create request and capture rollback
	t.Log("STEP 2: Creating request and capturing rollback")
	sess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(h.ProjectDir),
		testutil.WithAgent("restore-test-agent"),
		testutil.WithModel("test-model"),
	)

	req := testutil.MakeRequest(t, h.DB, sess,
		testutil.WithCommand("rm -rf "+testDir, h.ProjectDir, true),
		testutil.WithRisk(db.RiskTierDangerous),
		testutil.WithMinApprovals(1),
	)

	rollbackData, err := core.CaptureRollbackState(context.Background(), req, core.RollbackCaptureOptions{})
	if err != nil {
		t.Fatalf("CaptureRollbackState failed: %v", err)
	}
	t.Logf("  Rollback captured: %s", rollbackData.RollbackPath)

	// Step 3: Delete the files (simulate execution)
	t.Log("STEP 3: Deleting files")
	if err := os.RemoveAll(testDir); err != nil {
		t.Fatalf("Failed to remove test dir: %v", err)
	}

	// Verify files are gone
	if _, err := os.Stat(file1); !os.IsNotExist(err) {
		t.Error("file1.txt should be deleted")
	}
	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Error("file2.txt should be deleted")
	}
	t.Log("  Files deleted")

	// Step 4: Restore from rollback
	t.Log("STEP 4: Restoring from rollback")
	err = core.RestoreRollbackState(context.Background(), rollbackData, core.RollbackRestoreOptions{
		Force: true,
	})
	if err != nil {
		t.Fatalf("RestoreRollbackState failed: %v", err)
	}
	t.Log("  Rollback restored")

	// Step 5: Verify files restored
	t.Log("STEP 5: Verifying files restored")
	content1, err := os.ReadFile(file1)
	if err != nil {
		t.Errorf("file1.txt not restored: %v", err)
	} else if strings.TrimSpace(string(content1)) != "content1" {
		t.Errorf("file1.txt content wrong: got %q", string(content1))
	}

	content2, err := os.ReadFile(file2)
	if err != nil {
		t.Errorf("file2.txt not restored: %v", err)
	} else if strings.TrimSpace(string(content2)) != "content2" {
		t.Errorf("file2.txt content wrong: got %q", string(content2))
	}
	t.Log("  file1.txt restored with correct content")
	t.Log("  file2.txt restored with correct content")

	t.Log("=== PASS: TestFilesystemRollback_SuccessfulRestore ===")
}

// TestRollback_NonGitDirectory tests that commands in non-git directories
// still execute but appropriate rollback is captured.
func TestRollback_NonGitDirectory(t *testing.T) {
	h := testutil.NewHarness(t)

	t.Log("=== TestRollback_NonGitDirectory ===")
	t.Logf("ENV: temp_db=%s", h.DBPath)

	// Step 1: Create non-git directory with files
	t.Log("STEP 1: Creating non-git directory with files")
	nonGitDir := filepath.Join(h.ProjectDir, "non_git_test")
	if err := os.MkdirAll(nonGitDir, 0755); err != nil {
		t.Fatalf("Failed to create non-git dir: %v", err)
	}

	testFile := filepath.Join(nonGitDir, "data.txt")
	if err := os.WriteFile(testFile, []byte("test data\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	t.Logf("  Created: %s", testFile)

	// Step 2: Attempt to create git rollback (should fail gracefully)
	t.Log("STEP 2: Testing git command rollback in non-git dir")
	sess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(nonGitDir),
		testutil.WithAgent("non-git-agent"),
		testutil.WithModel("test-model"),
	)

	gitReq := testutil.MakeRequest(t, h.DB, sess,
		testutil.WithCommand("git status", nonGitDir, true),
		testutil.WithRisk(db.RiskTierCaution),
		testutil.WithMinApprovals(0),
	)

	rollbackData, err := core.CaptureRollbackState(context.Background(), gitReq, core.RollbackCaptureOptions{})
	// Git command in non-git dir should fail to capture
	if err != nil {
		t.Logf("  Expected: git rollback capture failed in non-git dir: %v", err)
	} else if rollbackData == nil {
		t.Log("  No rollback data (expected for non-git directory)")
	}

	// Step 3: Test filesystem rollback in non-git dir (should work)
	t.Log("STEP 3: Testing filesystem rollback in non-git dir")
	rmReq := testutil.MakeRequest(t, h.DB, sess,
		testutil.WithCommand("rm -rf "+testFile, nonGitDir, true),
		testutil.WithRisk(db.RiskTierDangerous),
		testutil.WithMinApprovals(1),
	)

	fsRollback, err := core.CaptureRollbackState(context.Background(), rmReq, core.RollbackCaptureOptions{})
	if err != nil {
		t.Fatalf("Filesystem rollback should work in non-git dir: %v", err)
	}
	if fsRollback == nil {
		t.Fatal("Expected filesystem rollback data")
	}
	if fsRollback.Kind != "filesystem" {
		t.Errorf("Expected kind=filesystem, got %s", fsRollback.Kind)
	}
	t.Logf("  Filesystem rollback captured: %s", fsRollback.RollbackPath)

	t.Log("=== PASS: TestRollback_NonGitDirectory ===")
}

// TestRollback_LoadFromDisk tests that rollback data can be loaded from disk.
func TestRollback_LoadFromDisk(t *testing.T) {
	h := testutil.NewHarness(t)

	t.Log("=== TestRollback_LoadFromDisk ===")
	t.Logf("ENV: temp_db=%s", h.DBPath)

	// Step 1: Create test files and capture rollback
	t.Log("STEP 1: Creating files and capturing rollback")
	testDir := filepath.Join(h.ProjectDir, "load_test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	testFile := filepath.Join(testDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	sess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(h.ProjectDir),
		testutil.WithAgent("load-test-agent"),
		testutil.WithModel("test-model"),
	)

	req := testutil.MakeRequest(t, h.DB, sess,
		testutil.WithCommand("rm -rf "+testDir, h.ProjectDir, true),
		testutil.WithRisk(db.RiskTierDangerous),
		testutil.WithMinApprovals(1),
	)

	originalData, err := core.CaptureRollbackState(context.Background(), req, core.RollbackCaptureOptions{})
	if err != nil {
		t.Fatalf("CaptureRollbackState failed: %v", err)
	}
	t.Logf("  Original rollback path: %s", originalData.RollbackPath)

	// Step 2: Load rollback data from disk
	t.Log("STEP 2: Loading rollback data from disk")
	loadedData, err := core.LoadRollbackData(originalData.RollbackPath)
	if err != nil {
		t.Fatalf("LoadRollbackData failed: %v", err)
	}

	// Step 3: Verify loaded data matches original
	t.Log("STEP 3: Verifying loaded data matches original")
	if loadedData.RequestID != originalData.RequestID {
		t.Errorf("RequestID mismatch: got %s, want %s", loadedData.RequestID, originalData.RequestID)
	}
	if loadedData.Kind != originalData.Kind {
		t.Errorf("Kind mismatch: got %s, want %s", loadedData.Kind, originalData.Kind)
	}
	if loadedData.CommandRaw != originalData.CommandRaw {
		t.Errorf("CommandRaw mismatch: got %s, want %s", loadedData.CommandRaw, originalData.CommandRaw)
	}
	t.Logf("  RequestID: %s", loadedData.RequestID)
	t.Logf("  Kind: %s", loadedData.Kind)
	t.Logf("  CapturedAt: %s", loadedData.CapturedAt)

	// Step 4: Use loaded data for restore
	t.Log("STEP 4: Testing restore with loaded data")
	if err := os.RemoveAll(testDir); err != nil {
		t.Fatalf("Failed to remove test dir: %v", err)
	}
	t.Log("  Files deleted")

	err = core.RestoreRollbackState(context.Background(), loadedData, core.RollbackRestoreOptions{
		Force: true,
	})
	if err != nil {
		t.Fatalf("RestoreRollbackState with loaded data failed: %v", err)
	}

	// Verify restore worked
	if _, err := os.Stat(testFile); err != nil {
		t.Errorf("File not restored: %v", err)
	}
	t.Log("  Restore with loaded data successful")

	t.Log("=== PASS: TestRollback_LoadFromDisk ===")
}

// Helper functions

// setupTestGitRepo creates a new git repository in the given directory.
func setupTestGitRepo(t *testing.T, baseDir string) string {
	t.Helper()

	repoDir := filepath.Join(baseDir, "test_repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config email failed: %v\n%s", err, out)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config name failed: %v\n%s", err, out)
	}

	return repoDir
}

// gitAdd adds a file to git staging.
func gitAdd(t *testing.T, repoDir, filename string) {
	t.Helper()
	cmd := exec.Command("git", "add", filename)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}
}

// gitCommit creates a commit with the given message.
func gitCommit(t *testing.T, repoDir, message string) {
	t.Helper()
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}
}

// gitHead returns the current HEAD commit hash.
func gitHead(t *testing.T, repoDir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD failed: %v", err)
	}
	return strings.TrimSpace(string(out))
}
