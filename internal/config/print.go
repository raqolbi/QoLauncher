package config

import (
	"fmt"
	"io"
)

const redacted = "***REDACTED***"

// Print writes the effective configuration to w with secrets redacted.
func (c *Config) Print(w io.Writer) {
	if c == nil {
		return
	}

	password := redacted
	if c.LogPassword == "" {
		password = ""
	}

	fmt.Fprintf(w, "APP_BINARY=%s\n", c.AppBinary)
	fmt.Fprintf(w, "APP_ARGS=%s\n", c.AppArgs)
	fmt.Fprintf(w, "APP_PORT=%d\n", c.AppPort)
	fmt.Fprintf(w, "APP_WORKDIR=%s\n", c.ResolvedWorkdir())
	fmt.Fprintf(w, "APP_RESTART_POLICY=%s\n", c.RestartPolicy)
	fmt.Fprintf(w, "APP_RESTART_DELAY=%s\n", c.RestartDelay)
	fmt.Fprintf(w, "APP_MAX_RESTART=%d\n", c.MaxRestart)
	fmt.Fprintf(w, "APP_RESTART_WINDOW=%s\n", c.RestartWindow)
	fmt.Fprintf(w, "APP_RESTART_BURST=%d\n", c.RestartBurst)
	fmt.Fprintf(w, "APP_SHUTDOWN_TIMEOUT=%s\n", c.ShutdownTimeout)
	fmt.Fprintf(w, "LOG_DIR=%s\n", c.LogDir)
	fmt.Fprintf(w, "LOG_PORT=%d\n", c.LogPort)
	fmt.Fprintf(w, "LOG_RETENTION_DAYS=%d\n", c.LogRetentionDays)
	fmt.Fprintf(w, "LOG_USERNAME=%s\n", c.LogUsername)
	fmt.Fprintf(w, "LOG_PASSWORD=%s\n", password)
	fmt.Fprintf(w, "LOG_LEVEL=%s\n", c.LogLevel)
	fmt.Fprintf(w, "TZ=%s\n", c.TZ)
	fmt.Fprintf(w, "VIEWER_ENABLED=%t\n", c.ViewerEnabled)
	fmt.Fprintf(w, "HEALTHCHECK_ENABLED=%t\n", c.HealthcheckEnabled)
	if c.HealthcheckEnabled {
		fmt.Fprintf(w, "HEALTHCHECK_TYPE=%s\n", c.HealthcheckType)
		fmt.Fprintf(w, "HEALTHCHECK_URL=%s\n", c.HealthcheckURL)
		fmt.Fprintf(w, "HEALTHCHECK_INTERVAL=%s\n", c.HealthcheckInterval)
		fmt.Fprintf(w, "HEALTHCHECK_TIMEOUT=%s\n", c.HealthcheckTimeout)
		fmt.Fprintf(w, "HEALTHCHECK_FAILURES=%d\n", c.HealthcheckFailures)
	}
}
