package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Validate checks configuration and returns an error with a documented message.
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	if strings.TrimSpace(c.AppBinary) == "" {
		return fmt.Errorf("APP_BINARY is required")
	}

	info, err := os.Stat(c.AppBinary)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("APP_BINARY not found")
		}
		return fmt.Errorf("APP_BINARY not found")
	}
	if info.IsDir() {
		return fmt.Errorf("APP_BINARY not found")
	}

	switch c.RestartPolicy {
	case RestartNever, RestartOnFailure, RestartAlways:
	default:
		return fmt.Errorf("invalid APP_RESTART_POLICY")
	}

	if c.AppPort != 0 && (c.AppPort < 1 || c.AppPort > 65535) {
		return fmt.Errorf("invalid APP_PORT")
	}

	if c.LogPort < 1 || c.LogPort > 65535 {
		return fmt.Errorf("invalid LOG_PORT")
	}

	if c.AppPort > 0 && c.LogPort == c.AppPort {
		return fmt.Errorf("LOG_PORT must differ from APP_PORT")
	}

	if c.RestartBurst < 1 {
		return fmt.Errorf("APP_RESTART_BURST must be >= 1")
	}

	if c.LogRetentionDays < 0 {
		return fmt.Errorf("invalid LOG_RETENTION_DAYS")
	}

	switch strings.ToLower(c.LogLevel) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid LOG_LEVEL")
	}

	if c.ViewerEnabled {
		if strings.TrimSpace(c.LogUsername) == "" || strings.TrimSpace(c.LogPassword) == "" {
			return fmt.Errorf("LOG_USERNAME and LOG_PASSWORD required when viewer enabled")
		}
	}

	if c.HealthcheckEnabled {
		if strings.TrimSpace(c.HealthcheckURL) == "" {
			return fmt.Errorf("HEALTHCHECK_URL required when healthcheck enabled")
		}
		if c.HealthcheckType != "http" {
			return fmt.Errorf("invalid HEALTHCHECK_TYPE")
		}
	}

	if err := ensureLogDirWritable(c.LogDir); err != nil {
		return fmt.Errorf("LOG_DIR is not writable")
	}

	return nil
}

func ensureLogDirWritable(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	testFile := filepath.Join(dir, ".write-test")
	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_ = f.Close()
	return os.Remove(testFile)
}

func dirOf(path string) string {
	dir := filepath.Dir(path)
	if dir == "." {
		return dir
	}
	return dir
}
