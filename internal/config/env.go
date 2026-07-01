package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// ApplyEnv overlays environment variables onto cfg.
func ApplyEnv(cfg *Config, environ []string) error {
	for _, entry := range environ {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if err := applyEnvKey(cfg, key, value); err != nil {
			return err
		}
	}
	return nil
}

func applyEnvKey(cfg *Config, key, value string) error {
	switch key {
	case "APP_BINARY":
		cfg.AppBinary = value
	case "APP_ARGS":
		cfg.AppArgs = value
	case "APP_PORT":
		port, err := strconv.Atoi(value)
		if err != nil {
			return errInvalid("APP_PORT")
		}
		cfg.AppPort = port
	case "APP_WORKDIR":
		cfg.AppWorkdir = value
	case "APP_RESTART_POLICY":
		cfg.RestartPolicy = RestartPolicy(value)
	case "APP_RESTART_DELAY":
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		cfg.RestartDelay = d
	case "APP_MAX_RESTART":
		n, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		cfg.MaxRestart = n
	case "APP_RESTART_WINDOW":
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		cfg.RestartWindow = d
	case "APP_RESTART_BURST":
		n, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		cfg.RestartBurst = n
	case "APP_SHUTDOWN_TIMEOUT":
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		cfg.ShutdownTimeout = d
	case "LOG_DIR":
		cfg.LogDir = value
	case "LOG_RETENTION_DAYS":
		n, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		cfg.LogRetentionDays = n
	case "LOG_LEVEL":
		cfg.LogLevel = value
	case "TZ":
		cfg.TZ = value
	case "LOG_PORT":
		port, err := strconv.Atoi(value)
		if err != nil {
			return errInvalid("LOG_PORT")
		}
		cfg.LogPort = port
	case "LOG_USERNAME":
		cfg.LogUsername = value
	case "LOG_PASSWORD":
		cfg.LogPassword = value
	case "VIEWER_ENABLED":
		b, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.ViewerEnabled = b
	case "HEALTHCHECK_ENABLED":
		b, err := parseBool(value)
		if err != nil {
			return err
		}
		cfg.HealthcheckEnabled = b
	case "HEALTHCHECK_TYPE":
		cfg.HealthcheckType = value
	case "HEALTHCHECK_URL":
		cfg.HealthcheckURL = value
	case "HEALTHCHECK_INTERVAL":
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		cfg.HealthcheckInterval = d
	case "HEALTHCHECK_TIMEOUT":
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		cfg.HealthcheckTimeout = d
	case "HEALTHCHECK_FAILURES":
		n, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		cfg.HealthcheckFailures = n
	}
	return nil
}

// ApplyEnvFromOS loads environment variables from the current process.
func ApplyEnvFromOS(cfg *Config) error {
	return ApplyEnv(cfg, os.Environ())
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, errInvalid(value)
	}
}

type invalidFieldError struct {
	field string
}

func (e invalidFieldError) Error() string {
	return "invalid " + e.field
}

func errInvalid(field string) error {
	return invalidFieldError{field: field}
}
