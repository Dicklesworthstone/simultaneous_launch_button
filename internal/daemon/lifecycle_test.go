package daemon

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/log"
)

func TestRunDaemon_SocketCreationFailure(t *testing.T) {
	// Create a file where the socket directory should be to force mkdir error
	tmp := t.TempDir()
	socketDir := filepath.Join(tmp, "socket")
	if err := os.WriteFile(socketDir, []byte("blocker"), 0600); err != nil {
		t.Fatal(err)
	}

	socketPath := filepath.Join(socketDir, "slb.sock")
	pidFile := filepath.Join(tmp, "slb.pid")

	logger := log.New(os.Stdout)
	ctx := context.Background()

	err := RunDaemon(ctx, ServerOptions{
		SocketPath: socketPath,
		PIDFile:    pidFile,
		Logger:     logger,
	})

	if err == nil {
		t.Error("expected error when socket directory cannot be created")
	}
	if !strings.Contains(err.Error(), "creating socket directory") {
		t.Errorf("expected socket directory creation error, got: %v", err)
	}
}

func TestRunDaemon_IPCServerFailure(t *testing.T) {
	tmp := t.TempDir()
	// Create a directory where the socket should be to force listen error
	// (net.Listen("unix", dir) usually fails)
	socketPath := filepath.Join(tmp, "slb.sock")
	if err := os.MkdirAll(socketPath, 0700); err != nil {
		t.Fatal(err)
	}

	pidFile := filepath.Join(tmp, "slb.pid")

	logger := log.New(os.Stdout)
	ctx := context.Background()

	err := RunDaemon(ctx, ServerOptions{
		SocketPath: socketPath,
		PIDFile:    pidFile,
		Logger:     logger,
	})

	if err == nil {
		t.Error("expected error when ipc server creation fails")
	}
}

func TestStopDaemon_Timeout(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "test.pid")

	// Start a dummy process that ignores signals or lives long enough
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Skipf("skipping test, cannot start sleep: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	if err := writePIDFile(pidFile, cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}

	// Expect timeout
	err := StopDaemonWithOptions(ServerOptions{
		PIDFile: pidFile,
	}, 10*time.Millisecond) // Short timeout

	if err == nil {
		t.Error("expected timeout error")
	}
	if !strings.Contains(err.Error(), "did not exit within") {
		t.Errorf("expected timeout message, got: %v", err)
	}
}

func TestStopDaemon_NoPIDFile(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "nonexistent.pid")

	err := StopDaemonWithOptions(ServerOptions{
		PIDFile: pidFile,
	}, 1*time.Second)

	if err == nil {
		t.Error("expected error when pid file missing")
	}
}

func TestStartDaemon_DaemonMode(t *testing.T) {
	// Simulate daemon mode enabled
	t.Setenv("SLB_DAEMON_MODE", "1")

	// We can't actually run RunDaemon fully without blocking or setting up complex context cancellation
	// But we can verify it calls RunDaemon by checking if it tries to use the socket path
	// or by using a mock logger/options.

	// Using a socket path that will fail quickly is a good way to verify it entered RunDaemon
	tmp := t.TempDir()
	socketDir := filepath.Join(tmp, "socket")
	if err := os.WriteFile(socketDir, []byte("blocker"), 0600); err != nil {
		t.Fatal(err)
	}
	socketPath := filepath.Join(socketDir, "slb.sock")
	pidFile := filepath.Join(tmp, "slb.pid")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := StartDaemonWithOptions(ctx, ServerOptions{
		SocketPath: socketPath,
		PIDFile:    pidFile,
	})

	// Should fail with directory creation error from RunDaemon
	if err == nil {
		t.Error("expected error from RunDaemon")
	}
	if !strings.Contains(err.Error(), "creating socket directory") {
		t.Errorf("expected RunDaemon execution error, got: %v", err)
	}
}

func TestRunDaemon_WithTCP(t *testing.T) {
	// Create project directory
	tmp := t.TempDir()
	slbDir := filepath.Join(tmp, ".slb")
	if err := os.MkdirAll(slbDir, 0700); err != nil {
		t.Fatal(err)
	}

	// Create config.toml enabling TCP
	configPath := filepath.Join(slbDir, "config.toml")
	configContent := `
[daemon]
tcp_addr = "127.0.0.1:0"
tcp_require_auth = false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Switch to project dir so RunDaemon picks up the config
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	socketPath := filepath.Join(slbDir, "slb.sock")
	pidFile := filepath.Join(slbDir, "slb.pid")
	logger := log.New(io.Discard)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start daemon in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunDaemon(ctx, ServerOptions{
			SocketPath: socketPath,
			PIDFile:    pidFile,
			Logger:     logger,
		})
	}()

	// Wait for PID file (daemon started)
	deadline := time.Now().Add(2 * time.Second)
	started := false
	for time.Now().Before(deadline) {
		if _, err := os.Stat(pidFile); err == nil {
			started = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !started {
		t.Fatal("timed out waiting for daemon to start")
	}

	// Verify socket exists
	if _, err := os.Stat(socketPath); err != nil {
		t.Errorf("socket not created: %v", err)
	}

	// Shutdown
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("RunDaemon exited with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for RunDaemon to exit")
	}
}
