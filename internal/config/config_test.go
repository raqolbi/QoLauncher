package config_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/raqolbi/qolauncher/internal/config"
)

func testBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app")
	if err := os.WriteFile(path, []byte{0x7f, 'E', 'L', 'F'}, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func validConfig(t *testing.T) *config.Config {
	t.Helper()
	bin := testBinary(t)
	logDir := t.TempDir()
	return &config.Config{
		AppBinary:           bin,
		AppPort:             8080,
		RestartPolicy:       config.RestartNever,
		RestartDelay:        3 * time.Second,
		RestartWindow:       60 * time.Second,
		RestartBurst:        10,
		ShutdownTimeout:     30 * time.Second,
		LogDir:              logDir,
		LogRetentionDays:    14,
		LogLevel:            "info",
		TZ:                  "UTC",
		LogPort:             8081,
		LogUsername:         "admin",
		LogPassword:         "secret",
		ViewerEnabled:       true,
		HealthcheckEnabled:  false,
		HealthcheckType:     "http",
		HealthcheckInterval: 30 * time.Second,
		HealthcheckTimeout:  5 * time.Second,
		HealthcheckFailures: 3,
	}
}

func TestDefaultValues(t *testing.T) {
	cfg := config.Default()
	if cfg.AppPort != 8080 {
		t.Fatalf("AppPort = %d, want 8080", cfg.AppPort)
	}
	if cfg.RestartPolicy != config.RestartNever {
		t.Fatalf("RestartPolicy = %q, want never", cfg.RestartPolicy)
	}
	if cfg.LogDir != "/var/log/qolauncher" {
		t.Fatalf("LogDir = %q", cfg.LogDir)
	}
	if cfg.ViewerEnabled != true {
		t.Fatal("ViewerEnabled should default true")
	}
	if cfg.HealthcheckEnabled != false {
		t.Fatal("HealthcheckEnabled should default false")
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	cfg := config.Default()
	env := []string{
		"APP_BINARY=/app/server",
		"APP_PORT=3000",
		"APP_RESTART_POLICY=on-failure",
		"VIEWER_ENABLED=false",
		"LOG_RETENTION_DAYS=30",
	}
	if err := config.ApplyEnv(cfg, env); err != nil {
		t.Fatal(err)
	}
	if cfg.AppBinary != "/app/server" {
		t.Fatalf("AppBinary = %q", cfg.AppBinary)
	}
	if cfg.AppPort != 3000 {
		t.Fatalf("AppPort = %d", cfg.AppPort)
	}
	if cfg.RestartPolicy != config.RestartOnFailure {
		t.Fatalf("RestartPolicy = %q", cfg.RestartPolicy)
	}
	if cfg.ViewerEnabled {
		t.Fatal("ViewerEnabled should be false")
	}
	if cfg.LogRetentionDays != 30 {
		t.Fatalf("LogRetentionDays = %d", cfg.LogRetentionDays)
	}
}

func TestCLIPriorityOverEnv(t *testing.T) {
	t.Setenv("APP_PORT", "3000")
	t.Setenv("APP_RESTART_POLICY", "always")

	cfg := config.Default()
	if err := config.ApplyEnvFromOS(cfg); err != nil {
		t.Fatal(err)
	}

	bin := testBinary(t)
	logDir := t.TempDir()
	_, err := config.ParseFlags([]string{
		"--binary", bin,
		"--app-port", "9090",
		"--restart-policy", "never",
		"--log-dir", logDir,
		"--log-username", "u",
		"--log-password", "p",
		"--viewer-enabled=false",
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.AppPort != 9090 {
		t.Fatalf("AppPort = %d, want CLI 9090", cfg.AppPort)
	}
	if cfg.RestartPolicy != config.RestartNever {
		t.Fatalf("RestartPolicy = %q, want never from CLI", cfg.RestartPolicy)
	}
}

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*config.Config)
		wantErr string
	}{
		{
			name: "missing binary",
			mutate: func(c *config.Config) {
				c.AppBinary = ""
			},
			wantErr: "APP_BINARY is required",
		},
		{
			name: "binary not found",
			mutate: func(c *config.Config) {
				c.AppBinary = "/no/such/binary"
			},
			wantErr: "APP_BINARY not found",
		},
		{
			name: "invalid restart policy",
			mutate: func(c *config.Config) {
				c.RestartPolicy = "sometimes"
			},
			wantErr: "invalid APP_RESTART_POLICY",
		},
		{
			name: "invalid app port",
			mutate: func(c *config.Config) {
				c.AppPort = 99999
			},
			wantErr: "invalid APP_PORT",
		},
		{
			name: "viewer auth required",
			mutate: func(c *config.Config) {
				c.LogUsername = ""
				c.LogPassword = ""
				c.ViewerEnabled = true
			},
			wantErr: "LOG_USERNAME and LOG_PASSWORD required when viewer enabled",
		},
		{
			name: "healthcheck url required",
			mutate: func(c *config.Config) {
				c.HealthcheckEnabled = true
				c.HealthcheckURL = ""
			},
			wantErr: "HEALTHCHECK_URL required when healthcheck enabled",
		},
		{
			name: "restart burst",
			mutate: func(c *config.Config) {
				c.RestartBurst = 0
			},
			wantErr: "APP_RESTART_BURST must be >= 1",
		},
		{
			name: "port conflict",
			mutate: func(c *config.Config) {
				c.AppPort = 8080
				c.LogPort = 8080
			},
			wantErr: "LOG_PORT must differ from APP_PORT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig(t)
			tt.mutate(cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestPrintRedactsPassword(t *testing.T) {
	cfg := validConfig(t)
	cfg.LogPassword = "super-secret"

	var buf bytes.Buffer
	cfg.Print(&buf)
	out := buf.String()
	if strings.Contains(out, "super-secret") {
		t.Fatal("password was not redacted")
	}
	if !strings.Contains(out, "LOG_PASSWORD=***REDACTED***") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestAppArgSlice(t *testing.T) {
	cfg := &config.Config{AppArgs: "--port 8080 --verbose"}
	args := cfg.AppArgSlice()
	if len(args) != 3 || args[0] != "--port" || args[1] != "8080" || args[2] != "--verbose" {
		t.Fatalf("args = %#v", args)
	}
}

func TestResolvedWorkdir(t *testing.T) {
	cfg := &config.Config{AppBinary: "/app/bin/server"}
	if got := cfg.ResolvedWorkdir(); got != "/app/bin" {
		t.Fatalf("workdir = %q", got)
	}
	cfg.AppWorkdir = "/custom"
	if got := cfg.ResolvedWorkdir(); got != "/custom" {
		t.Fatalf("workdir = %q", got)
	}
}

func TestChildEnvFiltersLauncherPrefix(t *testing.T) {
	env := []string{
		"APP_BINARY=/app/server",
		"LAUNCHER_DEBUG=1",
		"PATH=/usr/bin",
	}
	out := config.ChildEnv(env)
	if len(out) != 2 {
		t.Fatalf("ChildEnv len = %d, want 2", len(out))
	}
	for _, e := range out {
		if strings.HasPrefix(e, "LAUNCHER_") {
			t.Fatalf("LAUNCHER_ var leaked: %s", e)
		}
	}
}

func TestViewerDisabledSkipsAuth(t *testing.T) {
	cfg := validConfig(t)
	cfg.ViewerEnabled = false
	cfg.LogUsername = ""
	cfg.LogPassword = ""
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInvalidEnvAppPort(t *testing.T) {
	cfg := config.Default()
	err := config.ApplyEnv(cfg, []string{"APP_PORT=abc"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid APP_PORT") {
		t.Fatalf("error = %v", err)
	}
}
