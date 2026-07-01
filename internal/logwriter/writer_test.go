package logwriter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/raqolbi/qolauncher/internal/logwriter"
)

func TestWriteLineCreatesDailyFile(t *testing.T) {
	dir := t.TempDir()
	fixed := time.Date(2026, 7, 1, 15, 4, 5, 0, time.UTC)

	w, err := logwriter.New(dir, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	w.SetNowFunc(func() time.Time { return fixed })

	if err := w.WriteLine(logwriter.StreamStdout, "hello"); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteLine(logwriter.StreamStderr, "warn"); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "2026-07-01.log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "2026-07-01T15:04:05Z [stdout] hello") {
		t.Fatalf("stdout line missing: %q", content)
	}
	if !strings.Contains(content, "[stderr] warn") {
		t.Fatalf("stderr line missing: %q", content)
	}
}

func TestDailyRollover(t *testing.T) {
	dir := t.TempDir()
	day1 := time.Date(2026, 7, 1, 23, 59, 0, 0, time.UTC)
	day2 := day1.Add(2 * time.Minute)
	current := day1

	w, err := logwriter.New(dir, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	w.SetNowFunc(func() time.Time { return current })

	if err := w.WriteLine(logwriter.StreamStdout, "day1"); err != nil {
		t.Fatal(err)
	}
	current = day2
	if err := w.WriteLine(logwriter.StreamStdout, "day2"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "2026-07-01.log")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "2026-07-02.log")); err != nil {
		t.Fatal(err)
	}
}
