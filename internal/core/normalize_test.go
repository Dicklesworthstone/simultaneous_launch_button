package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePathsInCommand(t *testing.T) {
	cwd := filepath.Join(string(os.PathSeparator), "tmp", "slb-test-cwd")

	t.Run("expands ./ paths", func(t *testing.T) {
		out := ResolvePathsInCommand("rm -rf ./build", cwd)
		want := filepath.Join(cwd, "build")
		if !strings.Contains(out, want) {
			t.Fatalf("output %q does not contain %q", out, want)
		}
	})

	t.Run("expands ../ paths", func(t *testing.T) {
		out := ResolvePathsInCommand("rm -rf ../secrets", cwd)
		want := filepath.Clean(filepath.Join(cwd, "..", "secrets"))
		if !strings.Contains(out, want) {
			t.Fatalf("output %q does not contain %q", out, want)
		}
	})

	t.Run("expands ~/ paths even when cwd empty", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			t.Skip("no home directory available")
		}

		out := ResolvePathsInCommand("rm -rf ~/build", "")
		want := filepath.Join(home, "build")
		if !strings.Contains(out, want) {
			t.Fatalf("output %q does not contain %q", out, want)
		}
	})
}

func TestNormalizeCommandEdgeCases(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		res := NormalizeCommand("")
		if res.Primary != "" || len(res.Segments) != 0 || res.IsCompound {
			t.Fatalf("got Primary=%q Segments=%v IsCompound=%v", res.Primary, res.Segments, res.IsCompound)
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		res := NormalizeCommand("   \t  ")
		if res.Primary != "" || len(res.Segments) != 0 || res.IsCompound {
			t.Fatalf("got Primary=%q Segments=%v IsCompound=%v", res.Primary, res.Segments, res.IsCompound)
		}
	})

	t.Run("very long command does not panic", func(t *testing.T) {
		long := "echo " + strings.Repeat("a", 10_000)
		res := NormalizeCommand(long)
		if res.Original == "" {
			t.Fatalf("expected Original to be set")
		}
	})

	t.Run("subshell detection", func(t *testing.T) {
		res := NormalizeCommand("echo $(rm -rf /tmp)")
		if !res.HasSubshell {
			t.Fatalf("expected HasSubshell=true")
		}
	})
}

func TestNormalizeCommandEnvAssignments(t *testing.T) {
	res := NormalizeCommand("env FOO=bar BAR=baz kubectl delete pod nginx-123")
	if res.Primary != "kubectl delete pod nginx-123" {
		t.Fatalf("Primary=%q, want %q", res.Primary, "kubectl delete pod nginx-123")
	}
	if len(res.StrippedWrappers) == 0 || res.StrippedWrappers[0] != "env" {
		t.Fatalf("StrippedWrappers=%v, want prefix [env ...]", res.StrippedWrappers)
	}
}
