package core

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Dicklesworthstone/slb/internal/db"
)

func TestComputeCommandHash(t *testing.T) {
	tests := []struct {
		name string
		spec db.CommandSpec
	}{
		{
			name: "basic command",
			spec: db.CommandSpec{
				Raw:   "rm -rf /tmp/test",
				Cwd:   "/home/user/project",
				Argv:  []string{"rm", "-rf", "/tmp/test"},
				Shell: false,
			},
		},
		{
			name: "shell command",
			spec: db.CommandSpec{
				Raw:   "echo hello && echo world",
				Cwd:   "/home/user",
				Shell: true,
			},
		},
		{
			name: "empty argv",
			spec: db.CommandSpec{
				Raw:   "ls",
				Cwd:   "/tmp",
				Argv:  nil,
				Shell: false,
			},
		},
		{
			name: "empty cwd",
			spec: db.CommandSpec{
				Raw:   "echo test",
				Cwd:   "",
				Argv:  []string{"echo", "test"},
				Shell: false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hash := db.ComputeCommandHash(tc.spec)

			// Hash should be non-empty
			if hash == "" {
				t.Error("ComputeCommandHash returned empty string")
			}

			// Hash should be 64 characters (SHA256 hex)
			if len(hash) != 64 {
				t.Errorf("ComputeCommandHash returned hash of length %d, want 64", len(hash))
			}

			// Hash should be hex characters only
			for _, c := range hash {
				if !strings.ContainsRune("0123456789abcdef", c) {
					t.Errorf("ComputeCommandHash returned non-hex character %q", c)
					break
				}
			}

			// Same input should produce same hash
			hash2 := db.ComputeCommandHash(tc.spec)
			if hash != hash2 {
				t.Errorf("ComputeCommandHash not deterministic: %q != %q", hash, hash2)
			}
		})
	}
}

func TestComputeCommandHashUniqueness(t *testing.T) {
	// Different specs should produce different hashes
	specs := []db.CommandSpec{
		{Raw: "ls", Cwd: "/tmp", Shell: false},
		{Raw: "ls", Cwd: "/home", Shell: false},    // different cwd
		{Raw: "ls -la", Cwd: "/tmp", Shell: false}, // different raw
		{Raw: "ls", Cwd: "/tmp", Shell: true},      // different shell
		{Raw: "ls", Cwd: "/tmp", Argv: []string{"ls"}, Shell: false}, // with argv
	}

	hashes := make(map[string]int)
	for i, spec := range specs {
		hash := db.ComputeCommandHash(spec)
		if prevIdx, exists := hashes[hash]; exists {
			t.Errorf("Specs %d and %d produced same hash %q", prevIdx, i, hash)
		}
		hashes[hash] = i
	}
}

func TestRunCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell execution tests use Unix commands")
	}

	t.Run("shell mode executes raw command", func(t *testing.T) {
		spec := &db.CommandSpec{
			Raw:   "echo 'hello world'",
			Shell: true,
		}
		ctx := context.Background()
		result, err := RunCommand(ctx, spec, "", nil)
		if err != nil {
			t.Fatalf("RunCommand error: %v", err)
		}
		if !strings.Contains(result.Output, "hello world") {
			t.Errorf("expected output to contain 'hello world', got %q", result.Output)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
	})

	t.Run("argv mode executes parsed command", func(t *testing.T) {
		spec := &db.CommandSpec{
			Raw:   "echo hello",
			Argv:  []string{"echo", "hello"},
			Shell: false,
		}
		ctx := context.Background()
		result, err := RunCommand(ctx, spec, "", nil)
		if err != nil {
			t.Fatalf("RunCommand error: %v", err)
		}
		if !strings.Contains(result.Output, "hello") {
			t.Errorf("expected output to contain 'hello', got %q", result.Output)
		}
	})

	t.Run("raw mode parses command when no argv", func(t *testing.T) {
		spec := &db.CommandSpec{
			Raw:   "echo hello",
			Shell: false,
		}
		ctx := context.Background()
		result, err := RunCommand(ctx, spec, "", nil)
		if err != nil {
			t.Fatalf("RunCommand error: %v", err)
		}
		if !strings.Contains(result.Output, "hello") {
			t.Errorf("expected output to contain 'hello', got %q", result.Output)
		}
	})

	t.Run("empty command returns error", func(t *testing.T) {
		spec := &db.CommandSpec{
			Raw:   "",
			Shell: false,
		}
		ctx := context.Background()
		_, err := RunCommand(ctx, spec, "", nil)
		if err == nil {
			t.Error("expected error for empty command")
		}
	})

	t.Run("writes to log file", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")

		spec := &db.CommandSpec{
			Raw:   "echo 'logged output'",
			Shell: true,
		}
		ctx := context.Background()
		_, err := RunCommand(ctx, spec, logPath, nil)
		if err != nil {
			t.Fatalf("RunCommand error: %v", err)
		}

		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("ReadFile error: %v", err)
		}
		if !strings.Contains(string(content), "logged output") {
			t.Errorf("expected log file to contain output, got %q", string(content))
		}
		if !strings.Contains(string(content), "SLB Command Execution") {
			t.Errorf("expected log file to contain header")
		}
	})

	t.Run("writes to stream writer", func(t *testing.T) {
		var buf bytes.Buffer
		spec := &db.CommandSpec{
			Raw:   "echo 'streamed'",
			Shell: true,
		}
		ctx := context.Background()
		_, err := RunCommand(ctx, spec, "", &buf)
		if err != nil {
			t.Fatalf("RunCommand error: %v", err)
		}
		if !strings.Contains(buf.String(), "streamed") {
			t.Errorf("expected stream buffer to contain 'streamed', got %q", buf.String())
		}
	})

	t.Run("sets working directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		spec := &db.CommandSpec{
			Raw:   "pwd",
			Shell: true,
			Cwd:   tmpDir,
		}
		ctx := context.Background()
		result, err := RunCommand(ctx, spec, "", nil)
		if err != nil {
			t.Fatalf("RunCommand error: %v", err)
		}
		if !strings.Contains(result.Output, tmpDir) {
			t.Errorf("expected output to contain %q, got %q", tmpDir, result.Output)
		}
	})

	t.Run("captures non-zero exit code", func(t *testing.T) {
		spec := &db.CommandSpec{
			Raw:   "exit 42",
			Shell: true,
		}
		ctx := context.Background()
		result, err := RunCommand(ctx, spec, "", nil)
		// Non-zero exit code should not return error
		if err != nil {
			t.Fatalf("RunCommand error: %v", err)
		}
		if result.ExitCode != 42 {
			t.Errorf("expected exit code 42, got %d", result.ExitCode)
		}
	})

	t.Run("handles context timeout", func(t *testing.T) {
		spec := &db.CommandSpec{
			Raw:   "sleep 10",
			Shell: true,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		result, err := RunCommand(ctx, spec, "", nil)
		// Should either return timeout error or non-zero exit code
		if err == nil && result.ExitCode == 0 {
			t.Error("expected timeout to cause error or non-zero exit")
		}
	})

	t.Run("invalid log path returns error", func(t *testing.T) {
		spec := &db.CommandSpec{
			Raw:   "echo test",
			Shell: true,
		}
		ctx := context.Background()
		// Path that can't be opened
		_, err := RunCommand(ctx, spec, "/nonexistent/directory/file.log", nil)
		if err == nil {
			t.Error("expected error for invalid log path")
		}
	})

	t.Run("records duration", func(t *testing.T) {
		spec := &db.CommandSpec{
			Raw:   "sleep 0.1",
			Shell: true,
		}
		ctx := context.Background()
		result, err := RunCommand(ctx, spec, "", nil)
		if err != nil {
			t.Fatalf("RunCommand error: %v", err)
		}
		if result.Duration < 50*time.Millisecond {
			t.Errorf("expected duration >= 50ms, got %v", result.Duration)
		}
	})
}
