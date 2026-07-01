package supervisor_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/raqolbi/qolauncher/internal/config"
	"github.com/raqolbi/qolauncher/internal/logger"
	"github.com/raqolbi/qolauncher/internal/logwriter"
	"github.com/raqolbi/qolauncher/internal/supervisor"
)

func testConfig(t *testing.T, binary, args string, policy config.RestartPolicy) *config.Config {
	t.Helper()
	logDir := t.TempDir()
	return &config.Config{
		AppBinary:           binary,
		AppArgs:             args,
		AppPort:             8080,
		RestartPolicy:       policy,
		RestartDelay:        0,
		RestartWindow:       60 * time.Second,
		RestartBurst:        10,
		MaxRestart:          0,
		ShutdownTimeout:     2 * time.Second,
		LogDir:              logDir,
		LogRetentionDays:    0,
		LogLevel:            "error",
		TZ:                  "UTC",
		LogPort:             8081,
		ViewerEnabled:       false,
		HealthcheckEnabled:  false,
		HealthcheckType:     "http",
		HealthcheckInterval: 30 * time.Second,
		HealthcheckTimeout:  5 * time.Second,
		HealthcheckFailures: 3,
	}
}

func runSupervisor(t *testing.T, cfg *config.Config) int {
	t.Helper()
	var buf bytes.Buffer
	log := logger.New("error", &buf)
	events := logger.NewSupervisor(log)

	writer, err := logwriter.New(cfg.LogDir, cfg.TZ)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()

	sup := supervisor.New(cfg, events, writer)
	return sup.Run(context.Background())
}

func TestSupervisorNeverSuccess(t *testing.T) {
	cfg := testConfig(t, "/bin/true", "", config.RestartNever)
	if code := runSupervisor(t, cfg); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestSupervisorNeverFailure(t *testing.T) {
	cfg := testConfig(t, "/bin/false", "", config.RestartNever)
	if code := runSupervisor(t, cfg); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func TestSupervisorOnFailureMaxRestart(t *testing.T) {
	cfg := testConfig(t, "/bin/false", "", config.RestartOnFailure)
	cfg.MaxRestart = 1
	cfg.RestartDelay = 0

	if code := runSupervisor(t, cfg); code != 1 {
		t.Fatalf("exit code = %d, want 1 (max restart)", code)
	}
}

func TestSupervisorCrashLoopGuard(t *testing.T) {
	cfg := testConfig(t, "/bin/false", "", config.RestartOnFailure)
	cfg.RestartBurst = 2
	cfg.RestartWindow = 60 * time.Second
	cfg.RestartDelay = 0

	if code := runSupervisor(t, cfg); code != 1 {
		t.Fatalf("exit code = %d, want 1 (crash loop)", code)
	}
}

func TestParseExit(t *testing.T) {
	res := supervisor.ParseExit(nil)
	if res.Code != 0 || res.Failed() {
		t.Fatalf("expected success exit")
	}
}

func TestChildEnvNoLauncherPrefix(t *testing.T) {
	env := config.ChildEnv([]string{"APP_BINARY=/x", "LAUNCHER_DEBUG=1", "PATH=/bin"})
	for _, e := range env {
		if len(e) >= 9 && e[:9] == "LAUNCHER_" {
			t.Fatalf("LAUNCHER_ leaked: %s", e)
		}
	}
}
