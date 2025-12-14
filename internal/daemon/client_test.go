package daemon

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Dicklesworthstone/slb/internal/config"
	"github.com/Dicklesworthstone/slb/internal/db"
	"github.com/charmbracelet/log"
)

func TestDefaultSocketPath(t *testing.T) {
	path := DefaultSocketPath()
	if path == "" {
		t.Error("DefaultSocketPath returned empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("DefaultSocketPath returned relative path: %s", path)
	}
	// Should start with tmp dir
	if !hasPrefix(path, os.TempDir()) {
		t.Errorf("DefaultSocketPath not in temp dir: %s", path)
	}
	// Should end with .sock
	if filepath.Ext(path) != ".sock" {
		t.Errorf("DefaultSocketPath doesn't end with .sock: %s", path)
	}
}

func TestDefaultPIDFile(t *testing.T) {
	path := DefaultPIDFile()
	if path == "" {
		t.Error("DefaultPIDFile returned empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("DefaultPIDFile returned relative path: %s", path)
	}
	// Should end with .pid
	if filepath.Ext(path) != ".pid" {
		t.Errorf("DefaultPIDFile doesn't end with .pid: %s", path)
	}
}

func TestNewClient(t *testing.T) {
	// Default client
	c := NewClient()
	if c.socketPath == "" {
		t.Error("Default socket path is empty")
	}
	if c.pidFile == "" {
		t.Error("Default PID file is empty")
	}

	// With custom options
	customSocket := "/tmp/test-slb.sock"
	customPID := "/tmp/test-slb.pid"
	c = NewClient(
		WithSocketPath(customSocket),
		WithPIDFile(customPID),
	)
	if c.socketPath != customSocket {
		t.Errorf("Socket path mismatch: got %s, want %s", c.socketPath, customSocket)
	}
	if c.pidFile != customPID {
		t.Errorf("PID file mismatch: got %s, want %s", c.pidFile, customPID)
	}
}

func TestDaemonStatus_String(t *testing.T) {
	tests := []struct {
		status   DaemonStatus
		expected string
	}{
		{DaemonRunning, "running"},
		{DaemonNotRunning, "not running"},
		{DaemonUnresponsive, "unresponsive"},
		{DaemonStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsDaemonRunning_NoPIDFile(t *testing.T) {
	c := NewClient(
		WithPIDFile("/nonexistent/path/to/slb.pid"),
		WithSocketPath("/nonexistent/path/to/slb.sock"),
	)
	if c.IsDaemonRunning() {
		t.Error("IsDaemonRunning should return false when PID file doesn't exist")
	}
	if c.GetStatus() != DaemonNotRunning {
		t.Errorf("GetStatus should return DaemonNotRunning, got %s", c.GetStatus())
	}
}

func TestIsDaemonRunning_StalePIDFile(t *testing.T) {
	// Create a temporary PID file with a non-existent PID
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")
	// Use a very high PID that likely doesn't exist
	err := os.WriteFile(pidFile, []byte("999999999"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test PID file: %v", err)
	}

	c := NewClient(
		WithPIDFile(pidFile),
		WithSocketPath(filepath.Join(tmpDir, "test.sock")),
	)

	if c.IsDaemonRunning() {
		t.Error("IsDaemonRunning should return false for stale PID file")
	}
}

func TestIsDaemonRunning_ProcessAliveNoSocket(t *testing.T) {
	// Create a PID file with our own PID (we know we're alive)
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")
	err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
	if err != nil {
		t.Fatalf("Failed to create test PID file: %v", err)
	}

	c := NewClient(
		WithPIDFile(pidFile),
		WithSocketPath(filepath.Join(tmpDir, "nonexistent.sock")),
	)

	// Process is alive but socket doesn't exist
	status := c.GetStatus()
	if status != DaemonUnresponsive {
		t.Errorf("GetStatus should return DaemonUnresponsive when process alive but no socket, got %s", status)
	}
}

func TestGetStatusInfo(t *testing.T) {
	c := NewClient(
		WithPIDFile("/nonexistent/slb.pid"),
		WithSocketPath("/nonexistent/slb.sock"),
	)

	info := c.GetStatusInfo()
	if info.Status != DaemonNotRunning {
		t.Errorf("Expected DaemonNotRunning, got %s", info.Status)
	}
	if info.PIDFile != "/nonexistent/slb.pid" {
		t.Errorf("PIDFile mismatch: %s", info.PIDFile)
	}
	if info.SocketPath != "/nonexistent/slb.sock" {
		t.Errorf("SocketPath mismatch: %s", info.SocketPath)
	}
	if info.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestWarningMessages(t *testing.T) {
	msg := WarningMessage()
	if msg == "" {
		t.Error("WarningMessage should not be empty")
	}
	if !containsSubstring(msg, "daemon") {
		t.Error("WarningMessage should mention daemon")
	}

	short := ShortWarning()
	if short == "" {
		t.Error("ShortWarning should not be empty")
	}
	if len(short) >= len(msg) {
		t.Error("ShortWarning should be shorter than WarningMessage")
	}
}

func TestResetWarningState(t *testing.T) {
	ResetWarningState()
	// Just verify it doesn't panic
}

func TestWithDaemonOrFallback(t *testing.T) {
	ResetWarningState()

	c := NewClient(
		WithPIDFile("/nonexistent/slb.pid"),
		WithSocketPath("/nonexistent/slb.sock"),
	)

	fnCalled := false
	fallbackCalled := false

	c.WithDaemonOrFallback(
		func() { fnCalled = true },
		func() { fallbackCalled = true },
	)

	if fnCalled {
		t.Error("Primary function should not be called when daemon is not running")
	}
	if !fallbackCalled {
		t.Error("Fallback should be called when daemon is not running")
	}
}

func TestWithDaemonOrFallbackErr(t *testing.T) {
	ResetWarningState()

	c := NewClient(
		WithPIDFile("/nonexistent/slb.pid"),
		WithSocketPath("/nonexistent/slb.sock"),
	)

	fnCalled := false
	fallbackCalled := false

	err := c.WithDaemonOrFallbackErr(
		func() error { fnCalled = true; return nil },
		func() error { fallbackCalled = true; return nil },
	)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if fnCalled {
		t.Error("Primary function should not be called when daemon is not running")
	}
	if !fallbackCalled {
		t.Error("Fallback should be called when daemon is not running")
	}
}

func TestMustHaveDaemon(t *testing.T) {
	c := NewClient(
		WithPIDFile("/nonexistent/slb.pid"),
		WithSocketPath("/nonexistent/slb.sock"),
	)

	err := c.MustHaveDaemon()
	if err == nil {
		t.Error("MustHaveDaemon should return error when daemon is not running")
	}
	if !containsSubstring(err.Error(), "daemon") {
		t.Error("Error message should mention daemon")
	}
}

func TestTryDaemon(t *testing.T) {
	c := NewClient(
		WithPIDFile("/nonexistent/slb.pid"),
		WithSocketPath("/nonexistent/slb.sock"),
	)

	fnCalled := false
	usedDaemon, err := c.TryDaemon(func() error {
		fnCalled = true
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if usedDaemon {
		t.Error("usedDaemon should be false when daemon is not running")
	}
	if fnCalled {
		t.Error("Function should not be called when daemon is not running")
	}
}

func TestGetFeatureAvailability_NoDaemon(t *testing.T) {
	c := NewClient(
		WithPIDFile("/nonexistent/slb.pid"),
		WithSocketPath("/nonexistent/slb.sock"),
	)

	features := c.GetFeatureAvailability()

	if features.RealTimeUpdates {
		t.Error("RealTimeUpdates should be false without daemon")
	}
	if features.DesktopNotifications {
		t.Error("DesktopNotifications should be false without daemon")
	}
	if features.AgentMailNotifications {
		t.Error("AgentMailNotifications should be false without daemon")
	}
	if features.FastIPC {
		t.Error("FastIPC should be false without daemon")
	}
	if !features.FilePolling {
		t.Error("FilePolling should always be true (fallback)")
	}
}

func TestConvenienceFunctions(t *testing.T) {
	// Reset default client
	defaultClient = nil

	// Just verify they don't panic
	_ = IsDaemonRunning()
	_ = GetStatus()
	_ = GetStatusInfo()
	_ = GetFeatureAvailability()

	called := false
	WithDaemonOrFallback(
		func() {},
		func() { called = true },
	)
	if !called {
		t.Error("Convenience WithDaemonOrFallback should work")
	}
}

func TestWithLogger_SetsLogger(t *testing.T) {
	logger := log.New(io.Discard)
	c := NewClient(WithLogger(logger))
	if c.logger != logger {
		t.Fatalf("expected logger to be set via WithLogger")
	}
}

func TestShowDegradedWarningQuiet_WritesOnce(t *testing.T) {
	ResetWarningState()

	out := captureStderr(t, func() {
		ShowDegradedWarningQuiet()
		ShowDegradedWarningQuiet()
	})

	if got := strings.Count(out, "Warning:"); got != 1 {
		t.Fatalf("expected warning once, got %d:\n%s", got, out)
	}
	if !strings.Contains(out, ShortWarning()) {
		t.Fatalf("expected output to include ShortWarning, got:\n%s", out)
	}
}

func TestClient_GetStatusInfo_SLBHostTCP(t *testing.T) {
	logger := log.New(io.Discard)

	srv, err := NewTCPServer(TCPServerOptions{
		Addr:        "127.0.0.1:0",
		RequireAuth: true,
		AllowedIPs:  []string{"127.0.0.1"},
		ValidateAuth: func(_ context.Context, sessionKey string) (bool, error) {
			return sessionKey == "good", nil
		},
	}, logger)
	if err != nil {
		t.Fatalf("NewTCPServer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		_ = srv.Stop()
	})
	go func() { _ = srv.Start(ctx) }()

	addr := srv.listener.Addr().String()
	t.Setenv("SLB_HOST", addr)
	t.Setenv("SLB_SESSION_KEY", "good")

	info := NewClient().GetStatusInfo()
	if info.Status != DaemonRunning {
		t.Fatalf("expected daemon running, got %s (%s)", info.Status, info.Message)
	}
	if info.SocketPath != addr {
		t.Fatalf("expected SocketPath=%q, got %q", addr, info.SocketPath)
	}
	if info.PIDFile != "" {
		t.Fatalf("expected PIDFile empty for TCP mode, got %q", info.PIDFile)
	}
}

func TestClient_GetStatusInfo_SLBHostFallbackToUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket tests not supported on windows")
	}

	t.Setenv("SLB_HOST", "127.0.0.1:0")
	t.Setenv("SLB_SESSION_KEY", "ignored")

	socketPath := filepath.Join(t.TempDir(), "ipc.sock")
	srv, err := NewIPCServer(socketPath, log.New(io.Discard))
	if err != nil {
		t.Fatalf("NewIPCServer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		_ = srv.Stop()
	})
	go func() { _ = srv.Start(ctx) }()

	pidFile := filepath.Join(t.TempDir(), "missing.pid")
	client := NewClient(
		WithSocketPath(socketPath),
		WithPIDFile(pidFile),
	)
	info := client.GetStatusInfo()
	if info.Status != DaemonRunning {
		t.Fatalf("expected daemon running via unix, got %s (%s)", info.Status, info.Message)
	}
	if !info.SocketAlive {
		t.Fatalf("expected socket_alive=true")
	}
	if info.SocketPath != socketPath {
		t.Fatalf("expected SocketPath=%q, got %q", socketPath, info.SocketPath)
	}
	if info.PIDFile != pidFile {
		t.Fatalf("expected PIDFile=%q, got %q", pidFile, info.PIDFile)
	}
	if !strings.Contains(info.Message, "using local unix socket") {
		t.Fatalf("expected fallback message, got: %s", info.Message)
	}
}

func TestDaemonHelpers_PIDFileAndProcessAlive(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "daemon.pid")

	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())+"\n"), 0644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	ok, pid := daemonRunning(ServerOptions{PIDFile: pidFile})
	if !ok || pid != os.Getpid() {
		t.Fatalf("expected daemonRunning true with current pid, got ok=%v pid=%d", ok, pid)
	}

	if !processAlive(os.Getpid()) {
		t.Fatalf("expected current process to be alive")
	}

	if _, err := readPIDFile(filepath.Join(tmp, "missing.pid")); err == nil {
		t.Fatalf("expected error for missing pid file")
	}
}

func TestStartDaemonWithOptions_DuplicatePIDFilePreventsStart(t *testing.T) {
	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "daemon.pid")

	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())+"\n"), 0644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Setenv(daemonModeEnv, "")

	err := StartDaemonWithOptions(ctx, ServerOptions{
		SocketPath: filepath.Join(tmp, "daemon.sock"),
		PIDFile:    pidFile,
		Logger:     log.New(io.Discard),
	})
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("expected already running error, got: %v", err)
	}
}

func TestStartDaemonWithOptions_DaemonModeRunsAndStops(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket tests not supported on windows")
	}

	tmp := t.TempDir()
	socketPath := filepath.Join(tmp, "daemon.sock")
	pidFile := filepath.Join(tmp, "daemon.pid")

	t.Setenv(daemonModeEnv, "1")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- StartDaemonWithOptions(ctx, ServerOptions{
			SocketPath: socketPath,
			PIDFile:    pidFile,
			Logger:     log.New(io.Discard),
		})
	}()

	// Wait for pid file to be created.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(pidFile); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("StartDaemonWithOptions (daemon mode) returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("daemon did not stop in time")
	}
}

func TestStopDaemonWithOptions_MissingPIDFileReturnsError(t *testing.T) {
	tmp := t.TempDir()
	err := StopDaemonWithOptions(ServerOptions{
		PIDFile: filepath.Join(tmp, "missing.pid"),
	}, 50*time.Millisecond)
	if err == nil {
		t.Fatalf("expected error for missing pid file")
	}
}

func TestDefaultServerOptions_NonEmpty(t *testing.T) {
	opts := DefaultServerOptions()
	if strings.TrimSpace(opts.SocketPath) == "" || strings.TrimSpace(opts.PIDFile) == "" {
		t.Fatalf("expected DefaultServerOptions to include socket + pid paths")
	}
}

func TestSendDesktopNotification_LinuxUsesNotifySendWhenPresent(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only notify-send behavior")
	}

	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	logPath := filepath.Join(tmp, "notify.log")
	t.Setenv("SLB_NOTIFY_LOG", logPath)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	notifySend := filepath.Join(binDir, "notify-send")
	script := "#!/bin/sh\nset -eu\necho \"$@\" >> \"${SLB_NOTIFY_LOG}\"\nexit 0\n"
	if err := os.WriteFile(notifySend, []byte(script), 0755); err != nil {
		t.Fatalf("write notify-send: %v", err)
	}

	if err := SendDesktopNotification("Title", "Message"); err != nil {
		t.Fatalf("SendDesktopNotification: %v", err)
	}

	// Also cover timeout notify() linux path (adds -u critical).
	if err := notify("T2", "B2"); err != nil {
		t.Fatalf("timeout notify: %v", err)
	}

	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(b), "Title") || !strings.Contains(string(b), "T2") {
		t.Fatalf("expected notify-send to be invoked, got:\n%s", string(b))
	}
}

func TestSendDesktopNotification_RequiresMessage(t *testing.T) {
	if err := SendDesktopNotification("Title", ""); err == nil {
		t.Fatalf("expected error for empty message")
	}
}

func TestRunNoOutput_ErrorIncludesCommand(t *testing.T) {
	tmp := t.TempDir()
	fail := filepath.Join(tmp, "fail.sh")
	if runtime.GOOS == "windows" {
		t.Skip("shell script helper not supported on windows")
	}
	script := "#!/bin/sh\necho \"nope\"; exit 42\n"
	if err := os.WriteFile(fail, []byte(script), 0755); err != nil {
		t.Fatalf("write fail script: %v", err)
	}

	err := runNoOutput(fail)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Fatalf("expected error to mention failure, got: %v", err)
	}
}

func TestEscapeAppleScript_EscapesQuotesBackslashesAndNewlines(t *testing.T) {
	in := "a\"b\\c\nx"
	out := escapeAppleScript(in)

	if strings.Contains(out, "\n") {
		t.Fatalf("expected no raw newlines after escaping, got %q", out)
	}
	for i := 0; i < len(out); i++ {
		if out[i] != '"' {
			continue
		}
		if i == 0 || out[i-1] != '\\' {
			t.Fatalf("expected quote to be escaped at index %d, got %q", i, out)
		}
	}
	if !strings.Contains(out, "\\n") {
		t.Fatalf("expected newlines to be escaped, got %q", out)
	}
	if !strings.Contains(out, "\\\\") {
		t.Fatalf("expected backslashes to be escaped, got %q", out)
	}
}

func TestTimeoutConfigs_DefaultAndFromConfig(t *testing.T) {
	def := DefaultTimeoutConfig()
	if def.CheckInterval != DefaultCheckInterval {
		t.Fatalf("expected DefaultCheckInterval, got %s", def.CheckInterval)
	}
	if def.Action != TimeoutActionEscalate {
		t.Fatalf("expected default action escalate, got %s", def.Action)
	}

	cfg := config.DefaultConfig()
	cfg.General.TimeoutAction = "not-a-real-action"
	cfg.Notifications.DesktopEnabled = false
	parsed := TimeoutConfigFromConfig(cfg)
	if parsed.Action != TimeoutActionEscalate {
		t.Fatalf("expected invalid action to default to escalate, got %s", parsed.Action)
	}
	if parsed.DesktopNotify {
		t.Fatalf("expected DesktopNotify=false from config")
	}

	cfg.General.TimeoutAction = string(TimeoutActionAutoReject)
	cfg.Notifications.DesktopEnabled = true
	parsed = TimeoutConfigFromConfig(cfg)
	if parsed.Action != TimeoutActionAutoReject {
		t.Fatalf("expected action auto_reject, got %s", parsed.Action)
	}
	if !parsed.DesktopNotify {
		t.Fatalf("expected DesktopNotify=true from config")
	}
}

func TestStartTimeoutChecker_StartsAndStops(t *testing.T) {
	database := openTestDB(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler, err := StartTimeoutChecker(ctx, database, log.New(io.Discard))
	if err != nil {
		t.Fatalf("StartTimeoutChecker: %v", err)
	}
	if !handler.IsRunning() {
		t.Fatalf("expected handler running")
	}
	handler.Stop()
}

func TestWatcher_ErrorsChannelAndSendError(t *testing.T) {
	var w *Watcher
	ch := w.Errors()
	if _, ok := <-ch; ok {
		t.Fatalf("expected closed channel from nil watcher")
	}

	w2 := &Watcher{
		errors: make(chan error, 1),
		logger: log.New(io.Discard),
	}
	w2.sendError(nil)

	want := errors.New("boom")
	w2.sendError(want)
	select {
	case err := <-w2.errors:
		if err == nil || err.Error() != want.Error() {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for error")
	}

	// Fill channel and ensure we hit the drop path (should not block).
	w2.errors <- errors.New("full")
	w2.sendError(errors.New("dropped"))
}

func TestStartDaemonWithOptions_ForkPath_WritesPIDFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process spawning not supported in this test on windows")
	}

	truePath, err := exec.LookPath("true")
	if err != nil {
		t.Skip("true not available")
	}

	origArgs := append([]string(nil), os.Args...)
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{truePath}

	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "daemon.pid")

	t.Setenv(daemonModeEnv, "")

	if err := StartDaemonWithOptions(context.Background(), ServerOptions{
		SocketPath: filepath.Join(tmp, "daemon.sock"),
		PIDFile:    pidFile,
		Logger:     log.New(io.Discard),
	}); err != nil {
		t.Fatalf("StartDaemonWithOptions: %v", err)
	}

	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read pid file: %v", err)
	}
	if strings.TrimSpace(string(b)) == "" {
		t.Fatalf("expected pid file to contain pid")
	}
}

func TestStopDaemonWithOptions_StopsProcessAndRemovesPIDFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix signal tests not supported on windows")
	}

	sleepPath, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep not available")
	}

	cmd := exec.Command(sleepPath, "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})

	tmp := t.TempDir()
	pidFile := filepath.Join(tmp, "daemon.pid")
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0644); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	if err := StopDaemonWithOptions(ServerOptions{PIDFile: pidFile}, 2*time.Second); err != nil {
		t.Fatalf("StopDaemonWithOptions: %v", err)
	}
	select {
	case <-waitCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected child process to be reaped")
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatalf("expected pid file removed after stop")
	}
}

func TestRunDaemon_StartsTCPListenerFromConfigAndValidatesAuth(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket tests not supported on windows")
	}

	// Pick an available port, then release it so RunDaemon can bind.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	project := t.TempDir()
	slbDir := filepath.Join(project, ".slb")
	if err := os.MkdirAll(slbDir, 0755); err != nil {
		t.Fatalf("mkdir .slb: %v", err)
	}

	// Configure TCP listener.
	cfgPath := filepath.Join(slbDir, "config.toml")
	cfgToml := "[daemon]\n" +
		"tcp_addr = \"" + addr + "\"\n" +
		"tcp_require_auth = true\n" +
		"tcp_allowed_ips = [\"127.0.0.1\"]\n" +
		"\n" +
		"[notifications]\n" +
		"desktop_enabled = false\n"
	if err := os.WriteFile(cfgPath, []byte(cfgToml), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Create project DB with an active session matching the auth key.
	dbPath := filepath.Join(slbDir, "state.db")
	dbConn, err := db.OpenAndMigrate(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { dbConn.Close() })

	session := &db.Session{
		ID:          "sess-1",
		AgentName:   "Agent",
		Program:     "test",
		Model:       "test",
		ProjectPath: project,
		SessionKey:  "good",
	}
	if err := dbConn.CreateSession(session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Run daemon inside the temp project directory so it loads the project config.
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(project); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })

	socketPath := filepath.Join(project, "daemon.sock")
	pidFile := filepath.Join(project, "daemon.pid")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- RunDaemon(ctx, ServerOptions{
			SocketPath: socketPath,
			PIDFile:    pidFile,
			Logger:     log.New(io.Discard),
		})
	}()

	// Wait for TCP ping to succeed.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		pctx, pcancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		err := pingDaemonTCP(pctx, addr, "good")
		pcancel()
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunDaemon: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("daemon did not stop in time")
	}
}

func TestTimeoutHandler_DesktopNotificationHelpers(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only notify-send behavior")
	}

	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	notifySend := filepath.Join(binDir, "notify-send")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(notifySend, []byte(script), 0755); err != nil {
		t.Fatalf("write notify-send: %v", err)
	}

	h := &TimeoutHandler{logger: log.New(io.Discard)}
	req := &db.Request{
		ID:             "req-12345678",
		RequestorAgent: "Agent",
		RiskTier:       db.RiskTierCritical,
		Command:        db.CommandSpec{Raw: "rm -rf /tmp/x"},
	}

	h.sendDesktopNotification(req)
	h.sendAutoApproveWarning(req)
}

func TestExtractRemoteIP_ParsesTCPAddrAndHostPort(t *testing.T) {
	ip, err := extractRemoteIP(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234})
	if err != nil {
		t.Fatalf("extractRemoteIP TCPAddr: %v", err)
	}
	if ip == nil || ip.String() != "127.0.0.1" {
		t.Fatalf("unexpected ip: %v", ip)
	}

	ip, err = extractRemoteIP(stringAddr("127.0.0.1:5678"))
	if err != nil {
		t.Fatalf("extractRemoteIP hostport: %v", err)
	}
	if ip == nil || ip.String() != "127.0.0.1" {
		t.Fatalf("unexpected ip: %v", ip)
	}

	if _, err := extractRemoteIP(nil); err == nil {
		t.Fatalf("expected error for nil addr")
	}
}

func TestWatcher_Events_NilAndNonNil(t *testing.T) {
	var w *Watcher
	ch := w.Events()
	if _, ok := <-ch; ok {
		t.Fatalf("expected closed channel from nil watcher")
	}

	w2 := &Watcher{events: make(chan WatchEvent, 1)}
	if w2.Events() != w2.events {
		t.Fatalf("expected Events to return the internal channel")
	}
}

func TestNewWatcher_RequiresProjectPath(t *testing.T) {
	if _, err := NewWatcher(""); err == nil {
		t.Fatalf("expected error for empty project path")
	}
}

type stringAddr string

func (s stringAddr) Network() string { return "tcp" }
func (s stringAddr) String() string  { return string(s) }

func openTestDB(t *testing.T) *db.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "state.db")
	dbConn, err := db.OpenAndMigrate(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { dbConn.Close() })
	return dbConn
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	defer func() {
		_ = w.Close()
		os.Stderr = old
	}()

	fn()

	_ = w.Close()
	os.Stderr = old

	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	return string(b)
}

// Helper functions
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			hasPrefix(s, substr) ||
			hasSuffix(s, substr) ||
			containsInner(s, substr))
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
