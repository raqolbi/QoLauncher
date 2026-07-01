package rotator_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/raqolbi/qolauncher/internal/rotator"
)

func writeLogFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSweepDeletesOldFiles(t *testing.T) {
	dir := t.TempDir()
	loc := time.UTC
	today := time.Now().In(loc)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)

	old1 := today.AddDate(0, 0, -30).Format("2006-01-02") + ".log"
	old2 := today.AddDate(0, 0, -20).Format("2006-01-02") + ".log"
	recent := today.Format("2006-01-02") + ".log"

	writeLogFile(t, dir, old1)
	writeLogFile(t, dir, old2)
	writeLogFile(t, dir, recent)
	writeLogFile(t, dir, "notes.txt")

	if err := rotator.Sweep(dir, 14, "UTC"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, old1)); !os.IsNotExist(err) {
		t.Fatal("old file should be deleted")
	}
	if _, err := os.Stat(filepath.Join(dir, old2)); !os.IsNotExist(err) {
		t.Fatal("old file should be deleted")
	}
	if _, err := os.Stat(filepath.Join(dir, recent)); err != nil {
		t.Fatal("recent file should remain")
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Fatal("non-log file should remain")
	}
}

func TestSweepRetentionZeroSkips(t *testing.T) {
	dir := t.TempDir()
	writeLogFile(t, dir, "2020-01-01.log")

	if err := rotator.Sweep(dir, 0, "UTC"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "2020-01-01.log")); err != nil {
		t.Fatal("file should remain when retention is 0")
	}
}

func TestStartPeriodicRunsSweep(t *testing.T) {
	dir := t.TempDir()
	name := time.Now().In(time.UTC).AddDate(-1, 0, 0).Format("2006-01-02") + ".log"
	writeLogFile(t, dir, name)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rotator.StartPeriodic(ctx, 20*time.Millisecond, dir, 1, "UTC")
	time.Sleep(60 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
		t.Fatal("periodic sweep should delete old file")
	}
}
