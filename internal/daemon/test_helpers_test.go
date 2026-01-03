package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// shortSocketDir creates a temp directory with a short path for Unix socket tests.
// macOS has a 104-byte limit on Unix socket paths, and t.TempDir() includes the
// full test name which can easily exceed this limit.
func shortSocketDir(t *testing.T) string {
	t.Helper()

	// Generate a short random suffix
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		t.Fatalf("generating random suffix: %v", err)
	}
	suffix := hex.EncodeToString(buf[:])

	// Use /tmp directly for shorter paths (macOS temp dir is very long)
	dir := filepath.Join("/tmp", "slb-test-"+suffix)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("creating short temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}
