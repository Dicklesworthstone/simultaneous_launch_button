package core

import (
	"strings"
	"testing"
)

func TestGetDryRunCommand(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantOK    bool
		wantParts []string
	}{
		{
			name:      "kubectl delete adds dry-run",
			in:        "kubectl delete deployment foo",
			wantOK:    true,
			wantParts: []string{"kubectl", "delete", "--dry-run=client", "-o", "yaml"},
		},
		{
			name:      "kubectl delete keeps existing dry-run",
			in:        "kubectl delete deployment foo --dry-run=client",
			wantOK:    true,
			wantParts: []string{"kubectl", "delete", "--dry-run=client"},
		},
		{
			name:      "terraform destroy becomes plan -destroy",
			in:        "terraform destroy",
			wantOK:    true,
			wantParts: []string{"terraform", "plan", "-destroy"},
		},
		{
			name:      "rm becomes ls listing",
			in:        "rm -rf ./build",
			wantOK:    true,
			wantParts: []string{"ls", "-la", "./build"},
		},
		{
			name:      "git reset --hard becomes diff",
			in:        "git reset --hard HEAD~5",
			wantOK:    true,
			wantParts: []string{"git", "diff", "HEAD~5..HEAD"},
		},
		{
			name:      "helm uninstall becomes get manifest",
			in:        "helm uninstall myrelease",
			wantOK:    true,
			wantParts: []string{"helm", "get", "manifest", "myrelease"},
		},
		{
			name:      "wrapper stripping still detects kubectl",
			in:        "sudo kubectl delete pod nginx-123",
			wantOK:    true,
			wantParts: []string{"kubectl", "delete", "--dry-run=client"},
		},
		{
			name:   "unsupported command",
			in:     "echo hello",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, ok := GetDryRunCommand(tt.in)
			if ok != tt.wantOK {
				t.Fatalf("ok=%v, want %v (out=%q)", ok, tt.wantOK, out)
			}
			if !ok {
				return
			}
			for _, part := range tt.wantParts {
				if !strings.Contains(out, part) {
					t.Fatalf("output %q does not contain %q", out, part)
				}
			}
		})
	}
}
