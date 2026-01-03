package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Dicklesworthstone/slb/internal/db"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func setupRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)

	repo := t.TempDir()
	if err := ensureGitRepo(repo); err != nil {
		t.Fatalf("ensureGitRepo: %v", err)
	}
	if err := ensureGitIdentity(repo); err != nil {
		t.Fatalf("ensureGitIdentity: %v", err)
	}

	// Initial commit so GetBranch returns an actual branch (not HEAD).
	if err := os.WriteFile(filepath.Join(repo, "README.txt"), []byte("hi\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := gitAdd(repo, "README.txt"); err != nil {
		t.Fatalf("gitAdd: %v", err)
	}
	if committed, err := gitCommitIfNeeded(repo, "init"); err != nil || !committed {
		t.Fatalf("gitCommitIfNeeded: committed=%v err=%v", committed, err)
	}

	return repo
}

func TestExpandUserPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if _, err := expandUserPath(""); err == nil {
		t.Fatalf("expected error for empty path")
	}

	got, err := expandUserPath("~")
	if err != nil {
		t.Fatalf("expandUserPath(~): %v", err)
	}
	if got != home {
		t.Fatalf("expandUserPath(~)=%q want %q", got, home)
	}

	got, err = expandUserPath("~/")
	if err != nil {
		t.Fatalf("expandUserPath(~/): %v", err)
	}
	if got != home {
		t.Fatalf("expandUserPath(~/)=%q want %q", got, home)
	}

	got, err = expandUserPath("~/x/y")
	if err != nil {
		t.Fatalf("expandUserPath(~/x/y): %v", err)
	}
	if got != filepath.Join(home, "x", "y") {
		t.Fatalf("expandUserPath(~/x/y)=%q", got)
	}

	got, err = expandUserPath("~\\x")
	if err != nil {
		t.Fatalf("expandUserPath(~\\\\x): %v", err)
	}
	if got != filepath.Join(home, "x") {
		t.Fatalf("expandUserPath(~\\\\x)=%q", got)
	}
}

func TestRunGit(t *testing.T) {
	requireGit(t)

	if _, err := runGit("", "status"); err == nil {
		t.Fatalf("expected error for empty repoPath")
	}

	repo := t.TempDir()
	if err := ensureGitRepo(repo); err != nil {
		t.Fatalf("ensureGitRepo: %v", err)
	}

	out, err := runGit(repo, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatalf("runGit: %v", err)
	}
	if strings.TrimSpace(out) != "true" {
		t.Fatalf("unexpected output: %q", out)
	}

	// Exercise the "no stderr/stdout message" error path.
	if _, err := runGit(repo, "config", "--get", "does.not.exist"); err == nil {
		t.Fatalf("expected error for missing config key")
	}

	if _, err := runGit(repo, "rev-parse", "--verify", "definitely-does-not-exist-123"); err == nil {
		t.Fatalf("expected error for invalid revision")
	}
}

func TestRepoHelpers(t *testing.T) {
	repo := setupRepo(t)

	sub := filepath.Join(repo, "subdir")
	if err := os.MkdirAll(sub, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if !IsRepo(sub) {
		t.Fatalf("expected IsRepo(sub)=true")
	}
	if IsRepo(t.TempDir()) {
		t.Fatalf("expected IsRepo(non-repo)=false")
	}

	root, err := GetRoot(sub)
	if err != nil {
		t.Fatalf("GetRoot: %v", err)
	}
	// Resolve symlinks for comparison (macOS /var -> /private/var)
	rootResolved, _ := filepath.EvalSymlinks(root)
	repoResolved, _ := filepath.EvalSymlinks(repo)
	if rootResolved != repoResolved {
		t.Fatalf("GetRoot=%q want %q", root, repo)
	}

	branch, err := GetBranch(sub)
	if err != nil {
		t.Fatalf("GetBranch: %v", err)
	}
	if strings.TrimSpace(branch) == "" || branch == "HEAD" {
		t.Fatalf("unexpected branch: %q", branch)
	}
}

func TestGetRootAndBranch_NonRepoErrors(t *testing.T) {
	requireGit(t)

	nonRepo := t.TempDir()
	if _, err := GetRoot(nonRepo); err == nil {
		t.Fatalf("expected GetRoot error for non-repo")
	}
	if _, err := GetBranch(nonRepo); err == nil {
		t.Fatalf("expected GetBranch error for non-repo")
	}
}

func TestInstallHook(t *testing.T) {
	repo := setupRepo(t)

	if err := InstallHook(repo); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	hookPath := filepath.Join(repo, ".git", "hooks", "pre-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if !strings.Contains(string(data), "slb hook pre-commit") {
		t.Fatalf("unexpected hook content: %q", string(data))
	}
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("stat hook: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("expected hook to be executable; mode=%v", info.Mode().Perm())
	}
}

func TestInstallHook_NonRepoErrors(t *testing.T) {
	nonRepo := t.TempDir()
	if err := InstallHook(nonRepo); err == nil {
		t.Fatalf("expected error when installing hook outside git repo")
	}
}

func TestStagingAndCommitHelpers(t *testing.T) {
	repo := t.TempDir()
	requireGit(t)

	if err := ensureGitRepo(repo); err != nil {
		t.Fatalf("ensureGitRepo: %v", err)
	}
	if err := ensureGitIdentity(repo); err != nil {
		t.Fatalf("ensureGitIdentity: %v", err)
	}

	if _, err := runGit(repo, "config", "--get", "user.name"); err != nil {
		t.Fatalf("expected user.name configured: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := gitAdd(repo, "a.txt"); err != nil {
		t.Fatalf("gitAdd: %v", err)
	}
	hasChanges, err := stagedChangesExist(repo)
	if err != nil {
		t.Fatalf("stagedChangesExist: %v", err)
	}
	if !hasChanges {
		t.Fatalf("expected staged changes")
	}

	if _, err := gitCommitIfNeeded(repo, ""); err == nil {
		t.Fatalf("expected error for empty commit message")
	}

	committed, err := gitCommitIfNeeded(repo, "add a.txt")
	if err != nil {
		t.Fatalf("gitCommitIfNeeded: %v", err)
	}
	if !committed {
		t.Fatalf("expected commit")
	}

	committed, err = gitCommitIfNeeded(repo, "no changes")
	if err != nil {
		t.Fatalf("gitCommitIfNeeded: %v", err)
	}
	if committed {
		t.Fatalf("expected committed=false when no staged changes")
	}
}

func TestEnsureGitRepo_IdempotentAndGitAddNoPaths(t *testing.T) {
	repo := setupRepo(t)

	if err := ensureGitRepo(repo); err != nil {
		t.Fatalf("ensureGitRepo second call: %v", err)
	}
	if err := gitAdd(repo); err != nil {
		t.Fatalf("gitAdd no paths: %v", err)
	}
}

func TestEnsureGitIdentity_EmptyPathErrors(t *testing.T) {
	if err := ensureGitIdentity(""); err == nil {
		t.Fatalf("expected error for empty repoPath")
	}
}

func TestGitCommitIfNeeded_ErrorPaths(t *testing.T) {
	requireGit(t)

	// Non-repo should error during staged changes check.
	nonRepo := t.TempDir()
	if _, err := gitCommitIfNeeded(nonRepo, "msg"); err == nil {
		t.Fatalf("expected error for non-repo")
	}

	// Repo with staged changes but failing pre-commit hook should error on commit.
	repo := t.TempDir()
	if err := ensureGitRepo(repo); err != nil {
		t.Fatalf("ensureGitRepo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "b.txt"), []byte("b\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := gitAdd(repo, "b.txt"); err != nil {
		t.Fatalf("gitAdd: %v", err)
	}

	hookPath := filepath.Join(repo, ".git", "hooks", "pre-commit")
	if err := os.MkdirAll(filepath.Dir(hookPath), 0750); err != nil {
		t.Fatalf("mkdir hooks dir: %v", err)
	}
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatalf("write pre-commit hook: %v", err)
	}

	if _, err := gitCommitIfNeeded(repo, "blocked by hook"); err == nil {
		t.Fatalf("expected commit error due to pre-commit hook")
	}
}

func TestStagedChangesExist_NonRepoErrors(t *testing.T) {
	requireGit(t)

	nonRepo := t.TempDir()
	if _, err := stagedChangesExist(nonRepo); err == nil {
		t.Fatalf("expected error for non-repo")
	}
}

func TestHistoryRepo_InitAndCommits(t *testing.T) {
	repoPath := t.TempDir()
	requireGit(t)

	repo := &HistoryRepo{Path: repoPath}
	if err := repo.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	for _, dir := range []string{"requests", "reviews", "executions", "patterns"} {
		if _, err := os.Stat(filepath.Join(repoPath, dir)); err != nil {
			t.Fatalf("expected %s dir: %v", dir, err)
		}
	}

	if name, err := runGit(repoPath, "config", "--get", "user.name"); err != nil || name != defaultHistoryAuthorName {
		t.Fatalf("expected history author name=%q got %q err=%v", defaultHistoryAuthorName, name, err)
	}

	when := time.Date(2025, time.January, 2, 3, 4, 5, 0, time.UTC)

	req := &db.Request{
		ID:       "req-1",
		RiskTier: db.RiskTierDangerous,
		Command: db.CommandSpec{
			Raw:               "rm -rf build",
			ContainsSensitive: true,
			DisplayRedacted:   "rm -rf [REDACTED]",
		},
		CreatedAt: when,
	}
	committed, abs, err := repo.CommitRequest(req)
	if err != nil {
		t.Fatalf("CommitRequest: %v", err)
	}
	if !committed {
		t.Fatalf("expected commit")
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("expected request file: %v", err)
	}

	rev := &db.Review{
		ID:            "rev-1",
		RequestID:     req.ID,
		Decision:      db.DecisionApprove,
		CreatedAt:     when,
		Comments:      "ok",
		Signature:     "sig",
		ReviewerAgent: "Agent",
		ReviewerModel: "Model",
	}
	committed, abs, err = repo.CommitReview(rev)
	if err != nil {
		t.Fatalf("CommitReview: %v", err)
	}
	if !committed {
		t.Fatalf("expected commit")
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("expected review file: %v", err)
	}

	exit := 0
	execInfo := &db.Execution{
		ExecutedAt: &when,
		ExitCode:   &exit,
	}
	committed, abs, err = repo.CommitExecution(req.ID, execInfo)
	if err != nil {
		t.Fatalf("CommitExecution: %v", err)
	}
	if !committed {
		t.Fatalf("expected commit")
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("expected execution file: %v", err)
	}

	// Idempotency: same request content should not create a new commit.
	committed, _, err = repo.CommitRequest(req)
	if err != nil {
		t.Fatalf("CommitRequest second time: %v", err)
	}
	if committed {
		t.Fatalf("expected committed=false when no diff")
	}

	// CreatedAt/ExecutedAt fallbacks (zero times).
	req2 := &db.Request{ID: "req-2", RiskTier: db.RiskTierCaution, Command: db.CommandSpec{Raw: "echo hi"}}
	if committed, _, err := repo.CommitRequest(req2); err != nil || !committed {
		t.Fatalf("CommitRequest zero time: committed=%v err=%v", committed, err)
	}
	rev2 := &db.Review{ID: "rev-2", RequestID: req2.ID, Decision: db.DecisionReject}
	if committed, _, err := repo.CommitReview(rev2); err != nil || !committed {
		t.Fatalf("CommitReview zero time: committed=%v err=%v", committed, err)
	}
	exec2 := &db.Execution{ExitCode: &exit}
	if committed, _, err := repo.CommitExecution(req2.ID, exec2); err != nil || !committed {
		t.Fatalf("CommitExecution zero time: committed=%v err=%v", committed, err)
	}
}

func TestInitHistoryRepo(t *testing.T) {
	requireGit(t)

	path := t.TempDir()
	if err := InitHistoryRepo(path); err != nil {
		t.Fatalf("InitHistoryRepo: %v", err)
	}
}

func TestInitHistoryRepo_EmptyPathErrors(t *testing.T) {
	if err := InitHistoryRepo(""); err == nil {
		t.Fatalf("expected error for empty path")
	}
}

func TestHistoryRepo_ErrorCases(t *testing.T) {
	requireGit(t)

	var nilRepo *HistoryRepo
	if err := nilRepo.Init(); err == nil {
		t.Fatalf("expected error for nil history repo")
	}

	repo := &HistoryRepo{}
	if err := repo.Init(); err == nil {
		t.Fatalf("expected error for empty history repo path")
	}

	// Init should fail if the "repo path" points to an existing file.
	fileRoot := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(fileRoot, []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	repo = &HistoryRepo{Path: fileRoot}
	if err := repo.Init(); err == nil {
		t.Fatalf("expected error for file history repo path")
	}

	// Commit helpers should surface Init() errors (receiver has no Path).
	repo = &HistoryRepo{}
	req := &db.Request{ID: "req", RiskTier: db.RiskTierDangerous, Command: db.CommandSpec{Raw: "echo hi"}}
	if _, _, err := repo.CommitRequest(req); err == nil {
		t.Fatalf("expected CommitRequest error when Init fails")
	}
	rev := &db.Review{ID: "rev", RequestID: "req", Decision: db.DecisionApprove}
	if _, _, err := repo.CommitReview(rev); err == nil {
		t.Fatalf("expected CommitReview error when Init fails")
	}
	if _, _, err := repo.CommitExecution("req", &db.Execution{}); err == nil {
		t.Fatalf("expected CommitExecution error when Init fails")
	}

	repo.Path = t.TempDir()
	if _, _, err := repo.CommitRequest(nil); err == nil {
		t.Fatalf("expected error for nil request")
	}
	if _, _, err := repo.CommitReview(nil); err == nil {
		t.Fatalf("expected error for nil review")
	}
	if _, _, err := repo.CommitExecution("", &db.Execution{}); err == nil {
		t.Fatalf("expected error for empty requestID")
	}
	if _, _, err := repo.CommitExecution("req", nil); err == nil {
		t.Fatalf("expected error for nil execution")
	}

	if _, err := repo.writeJSON("", map[string]any{"a": 1}); err == nil {
		t.Fatalf("expected error for empty relPath")
	}
	if _, err := repo.writeJSON("bad.json", make(chan int)); err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestHistoryRepo_ConstructorsAndHelpers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if _, err := NewHistoryRepo(""); err == nil {
		t.Fatalf("expected error for empty path")
	}

	repo, err := NewHistoryRepo("~/history")
	if err != nil {
		t.Fatalf("NewHistoryRepo: %v", err)
	}
	if repo.Path != filepath.Join(home, "history") {
		t.Fatalf("unexpected expanded path: %q", repo.Path)
	}

	if got := yearMonthPath(time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC)); got != filepath.Join("2025", "03") {
		t.Fatalf("yearMonthPath=%q", got)
	}

	req := &db.Request{Command: db.CommandSpec{Raw: "raw"}}
	if got := requestCommandForDisplay(req); got != "raw" {
		t.Fatalf("requestCommandForDisplay=%q", got)
	}
	req.Command.ContainsSensitive = true
	if got := requestCommandForDisplay(req); got != "<redacted>" {
		t.Fatalf("requestCommandForDisplay=%q", got)
	}
	req.Command.DisplayRedacted = "safe"
	if got := requestCommandForDisplay(req); got != "safe" {
		t.Fatalf("requestCommandForDisplay=%q", got)
	}

	if got := requestCommandForDisplay(nil); got != "" {
		t.Fatalf("requestCommandForDisplay(nil)=%q", got)
	}

	req = &db.Request{Command: db.CommandSpec{Raw: "raw", DisplayRedacted: "display"}}
	if got := requestCommandForDisplay(req); got != "display" {
		t.Fatalf("requestCommandForDisplay(display)=%q", got)
	}

	if got := truncateForCommit("a\nb\rc", 4); strings.ContainsAny(got, "\n\r") {
		t.Fatalf("expected newlines removed, got %q", got)
	}
	if got := truncateForCommit("abcdef", 0); got != "" {
		t.Fatalf("expected empty for max<=0, got %q", got)
	}
	if got := truncateForCommit("abcdef", 3); got != "abc" {
		t.Fatalf("expected max<=3 to hard truncate, got %q", got)
	}
}
