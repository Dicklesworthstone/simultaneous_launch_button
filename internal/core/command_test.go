package core

import (
	"strings"
	"testing"

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
			hash := ComputeCommandHash(tc.spec)

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
			hash2 := ComputeCommandHash(tc.spec)
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
		hash := ComputeCommandHash(spec)
		if prevIdx, exists := hashes[hash]; exists {
			t.Errorf("Specs %d and %d produced same hash %q", prevIdx, i, hash)
		}
		hashes[hash] = i
	}
}
