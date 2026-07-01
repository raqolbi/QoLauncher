package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestRunHelp(t *testing.T) {
	out := captureStdout(func() {
		if code := run([]string{"--help"}); code != 0 {
			t.Fatalf("exit code = %d", code)
		}
	})
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("help output missing Usage: %q", out)
	}
}

func TestRunVersion(t *testing.T) {
	out := captureStdout(func() {
		if code := run([]string{"--version"}); code != 0 {
			t.Fatalf("exit code = %d", code)
		}
	})
	if !strings.Contains(out, "QoLauncher v") {
		t.Fatalf("version output = %q", out)
	}
}

func TestRunConfigValidation(t *testing.T) {
	if code := run([]string{"--config"}); code == 0 {
		t.Fatal("expected failure without APP_BINARY")
	}
}

func TestRunConfigWithBinary(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "app")
	if err := os.WriteFile(bin, []byte{0x7f, 'E', 'L', 'F'}, 0o755); err != nil {
		t.Fatal(err)
	}
	logDir := t.TempDir()

	out := captureStdout(func() {
		code := run([]string{
			"--config",
			"--binary", bin,
			"--log-dir", logDir,
			"--log-username", "admin",
			"--log-password", "secret",
			"--viewer-enabled=true",
		})
		if code != 0 {
			t.Fatalf("exit code = %d", code)
		}
	})

	if strings.Contains(out, "secret") {
		t.Fatal("password leaked in --config output")
	}
	if !strings.Contains(out, "LOG_PASSWORD=***REDACTED***") {
		t.Fatalf("output:\n%s", out)
	}
}

func TestVersionSet(t *testing.T) {
	if version == "" {
		t.Fatal("version must not be empty")
	}
}
