package capture_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/raqolbi/qolauncher/internal/capture"
	"github.com/raqolbi/qolauncher/internal/logwriter"
)

func TestCaptureStdoutStderr(t *testing.T) {
	dir := t.TempDir()
	w, err := logwriter.New(dir, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	w.SetNowFunc(func() time.Time {
		return time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	})

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	var wg sync.WaitGroup
	capture.Pipes(stdoutR, stderrR, w, &wg)

	go func() {
		_, _ = stdoutW.Write([]byte("line one\npartial"))
		_ = stdoutW.Close()
	}()
	go func() {
		_, _ = stderrW.Write([]byte("err line\n"))
		_ = stderrW.Close()
	}()

	wg.Wait()

	data, err := os.ReadFile(filepath.Join(dir, "2026-07-01.log"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "[stdout] line one") {
		t.Fatalf("stdout missing: %q", content)
	}
	if !strings.Contains(content, "[stdout] partial") {
		t.Fatalf("partial stdout missing: %q", content)
	}
	if !strings.Contains(content, "[stderr] err line") {
		t.Fatalf("stderr missing: %q", content)
	}
}

func TestConcurrentCaptureNoBrokenLines(t *testing.T) {
	dir := t.TempDir()
	w, err := logwriter.New(dir, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	w.SetNowFunc(func() time.Time { return time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC) })

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	var wg sync.WaitGroup
	capture.Pipes(stdoutR, stderrR, w, &wg)

	go func() {
		for i := 0; i < 50; i++ {
			if i%2 == 0 {
				_, _ = stdoutW.Write([]byte("out\n"))
			} else {
				_, _ = stderrW.Write([]byte("err\n"))
			}
		}
		_ = stdoutW.Close()
		_ = stderrW.Close()
	}()
	wg.Wait()

	data, err := os.ReadFile(filepath.Join(dir, "2026-07-01.log"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 50 {
		t.Fatalf("line count = %d, want 50", len(lines))
	}
	for _, line := range lines {
		if !strings.Contains(line, "[stdout]") && !strings.Contains(line, "[stderr]") {
			t.Fatalf("broken line: %q", line)
		}
	}
}
