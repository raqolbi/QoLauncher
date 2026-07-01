package logger_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/raqolbi/qolauncher/internal/logger"
)

func TestLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New("warn", &buf)

	log.Info("hidden", nil)
	log.Warn("visible", map[string]any{"pid": 1})

	out := buf.String()
	if strings.Contains(out, "hidden") {
		t.Fatal("info should be filtered")
	}
	if !strings.Contains(out, `msg="visible"`) {
		t.Fatalf("output = %q", out)
	}
}

func TestLoggerFormat(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New("info", &buf)
	log.Info("launcher started", map[string]any{"version": "0.1.0"})

	out := buf.String()
	if !strings.Contains(out, "level=info") {
		t.Fatalf("missing level: %q", out)
	}
	if !strings.Contains(out, `msg="launcher started"`) {
		t.Fatalf("missing msg: %q", out)
	}
	if !strings.Contains(out, "version=0.1.0") {
		t.Fatalf("missing version: %q", out)
	}
}

func TestSupervisorEvents(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New("info", &buf)
	sup := logger.NewSupervisor(log)

	sup.LauncherStarted("0.1.0", nil)
	sup.ApplicationStarted("/app/server", 42, 0)
	sup.ApplicationCrashed(42, 2, "")
	sup.RestartingApplication(0, 1, "on-failure")

	out := buf.String()
	for _, want := range []string{
		"launcher started",
		"application started",
		"application crashed",
		"restarting application",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
