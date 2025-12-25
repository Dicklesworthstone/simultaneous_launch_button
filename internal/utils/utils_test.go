package utils

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
)

func TestParseLevel(t *testing.T) {
	cases := []struct {
		in   string
		want log.Level
	}{
		{"debug", log.DebugLevel},
		{"INFO", log.InfoLevel},
		{"warn", log.WarnLevel},
		{"warning", log.WarnLevel},
		{"error", log.ErrorLevel},
		{"fatal", log.FatalLevel},
		{"unknown", log.InfoLevel},
	}

	for _, tc := range cases {
		if got := parseLevel(tc.in); got != tc.want {
			t.Fatalf("parseLevel(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestInitLogger_WritesOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := InitLogger(LoggerOptions{
		Level:           "debug",
		Output:          &buf,
		Prefix:          "test",
		ReportTimestamp: false,
	})

	logger.Info("hello", "k", "v")
	if !strings.Contains(buf.String(), "hello") {
		t.Fatalf("expected output to contain message; got %q", buf.String())
	}
}

func TestInitDefaultLogger_RespectsEnvOverride(t *testing.T) {
	t.Setenv("SLB_LOG_LEVEL", "debug")
	logger := InitDefaultLogger()
	if logger == nil {
		t.Fatalf("expected logger")
	}
}

func TestInitDaemonLogger_CreatesLogFileUnderHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	logger, err := InitDaemonLogger()
	if err != nil {
		t.Fatalf("InitDaemonLogger: %v", err)
	}
	if logger == nil {
		t.Fatalf("expected logger")
	}

	path := filepath.Join(home, ".slb", "daemon.log")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected daemon log file at %s: %v", path, err)
	}
}

func TestInitRequestLogger_CreatesLogFileUnderProject(t *testing.T) {
	projectDir := t.TempDir()

	logger, err := InitRequestLogger(projectDir, "request-1234567890")
	if err != nil {
		t.Fatalf("InitRequestLogger: %v", err)
	}
	if logger == nil {
		t.Fatalf("expected logger")
	}

	matches, err := filepath.Glob(filepath.Join(projectDir, ".slb", "logs", "*.log"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 log file, got %d: %#v", len(matches), matches)
	}
}

func TestDefaultLoggerWrappers(t *testing.T) {
	old := GetDefaultLogger()
	t.Cleanup(func() {
		SetDefaultLogger(old)
	})

	var buf bytes.Buffer
	logger := InitLogger(LoggerOptions{
		Level:           "debug",
		Output:          &buf,
		Prefix:          "wrapper",
		ReportTimestamp: false,
	})
	SetDefaultLogger(logger)

	Debug("debug-msg")
	Info("info-msg")
	Warn("warn-msg")
	Error("error-msg")
	_ = With("k", "v")
	_ = WithPrefix("p")

	out := buf.String()
	for _, want := range []string{"debug-msg", "info-msg", "warn-msg", "error-msg"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q; got %q", want, out)
		}
	}
}
