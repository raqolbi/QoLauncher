package config

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

// ParseFlags parses CLI arguments into cfg. Priority: CLI > existing cfg values (ENV/defaults).
// Special flags --help, --version, and --config are handled via Options.
func ParseFlags(args []string, cfg *Config) (opts Options, err error) {
	fs := flag.NewFlagSet("launcher", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.AppBinary, "binary", cfg.AppBinary, "Path to application binary")
	fs.StringVar(&cfg.AppArgs, "args", cfg.AppArgs, "Arguments for application")
	fs.IntVar(&cfg.AppPort, "app-port", cfg.AppPort, "Application port metadata")
	fs.StringVar(&cfg.AppWorkdir, "workdir", cfg.AppWorkdir, "Working directory for application")

	fs.StringVar((*string)(&cfg.RestartPolicy), "restart-policy", string(cfg.RestartPolicy), "Restart policy: never, on-failure, always")
	fs.DurationVar(&cfg.RestartDelay, "restart-delay", cfg.RestartDelay, "Delay before restart")
	fs.IntVar(&cfg.MaxRestart, "max-restart", cfg.MaxRestart, "Max lifetime restarts (0=unlimited)")
	fs.DurationVar(&cfg.RestartWindow, "restart-window", cfg.RestartWindow, "Crash loop sliding window")
	fs.IntVar(&cfg.RestartBurst, "restart-burst", cfg.RestartBurst, "Max restarts within window")
	fs.DurationVar(&cfg.ShutdownTimeout, "shutdown-timeout", cfg.ShutdownTimeout, "Graceful shutdown timeout")

	fs.StringVar(&cfg.LogDir, "log-dir", cfg.LogDir, "Log output directory")
	fs.IntVar(&cfg.LogRetentionDays, "log-retention-days", cfg.LogRetentionDays, "Log retention in days")
	fs.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Launcher log level")
	fs.StringVar(&cfg.TZ, "tz", cfg.TZ, "Timezone for log rotation")

	fs.IntVar(&cfg.LogPort, "viewer-port", cfg.LogPort, "Log viewer HTTP port")
	fs.StringVar(&cfg.LogUsername, "log-username", cfg.LogUsername, "Viewer Basic Auth username")
	fs.StringVar(&cfg.LogPassword, "log-password", cfg.LogPassword, "Viewer Basic Auth password")
	fs.BoolVar(&cfg.ViewerEnabled, "viewer-enabled", cfg.ViewerEnabled, "Enable log viewer")

	fs.BoolVar(&cfg.HealthcheckEnabled, "healthcheck-enabled", cfg.HealthcheckEnabled, "Enable health probe")
	fs.StringVar(&cfg.HealthcheckType, "healthcheck-type", cfg.HealthcheckType, "Health probe type")
	fs.StringVar(&cfg.HealthcheckURL, "healthcheck-url", cfg.HealthcheckURL, "Health check URL")
	fs.DurationVar(&cfg.HealthcheckInterval, "healthcheck-interval", cfg.HealthcheckInterval, "Health probe interval")
	fs.DurationVar(&cfg.HealthcheckTimeout, "healthcheck-timeout", cfg.HealthcheckTimeout, "Health probe timeout")
	fs.IntVar(&cfg.HealthcheckFailures, "healthcheck-failures", cfg.HealthcheckFailures, "Consecutive failures before restart")

	opts.showHelp = fs.Bool("help", false, "Show usage")
	opts.showVersion = fs.Bool("version", false, "Print version and exit")
	opts.showConfig = fs.Bool("config", false, "Print resolved config and exit")

	if err := fs.Parse(args); err != nil {
		return opts, err
	}

	opts.args = fs.Args()
	return opts, nil
}

// Options holds special CLI modes parsed from flags.
type Options struct {
	showHelp    *bool
	showVersion *bool
	showConfig  *bool
	args        []string
}

func (o Options) Help() bool    { return o.showHelp != nil && *o.showHelp }
func (o Options) Version() bool { return o.showVersion != nil && *o.showVersion }
func (o Options) Config() bool  { return o.showConfig != nil && *o.showConfig }

// Usage prints CLI help to w.
func Usage(w io.Writer) {
	fmt.Fprintln(w, "QoLauncher - Universal Docker runtime for Go binaries")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  launcher [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	flags := []struct {
		name, env, desc string
	}{
		{"--help", "", "Show usage"},
		{"--version", "", "Print version and exit"},
		{"--config", "", "Print resolved config and exit"},
		{"--binary", "APP_BINARY", "Path to application binary"},
		{"--args", "APP_ARGS", "Arguments for application"},
		{"--app-port", "APP_PORT", "Application port metadata"},
		{"--workdir", "APP_WORKDIR", "Working directory for application"},
		{"--restart-policy", "APP_RESTART_POLICY", "never, on-failure, always"},
		{"--restart-delay", "APP_RESTART_DELAY", "Delay before restart"},
		{"--max-restart", "APP_MAX_RESTART", "Max lifetime restarts (0=unlimited)"},
		{"--restart-window", "APP_RESTART_WINDOW", "Crash loop sliding window"},
		{"--restart-burst", "APP_RESTART_BURST", "Max restarts in window"},
		{"--shutdown-timeout", "APP_SHUTDOWN_TIMEOUT", "Graceful shutdown timeout"},
		{"--log-dir", "LOG_DIR", "Log output directory"},
		{"--viewer-port", "LOG_PORT", "Log viewer HTTP port"},
		{"--log-retention-days", "LOG_RETENTION_DAYS", "Log retention in days"},
		{"--log-username", "LOG_USERNAME", "Viewer Basic Auth username"},
		{"--log-password", "LOG_PASSWORD", "Viewer Basic Auth password"},
		{"--log-level", "LOG_LEVEL", "Launcher log level"},
		{"--tz", "TZ", "Timezone for log rotation"},
		{"--viewer-enabled", "VIEWER_ENABLED", "Enable log viewer"},
		{"--healthcheck-enabled", "HEALTHCHECK_ENABLED", "Enable health probe"},
		{"--healthcheck-type", "HEALTHCHECK_TYPE", "Probe type (http)"},
		{"--healthcheck-url", "HEALTHCHECK_URL", "Health check URL"},
		{"--healthcheck-interval", "HEALTHCHECK_INTERVAL", "Probe interval"},
		{"--healthcheck-timeout", "HEALTHCHECK_TIMEOUT", "Probe timeout"},
		{"--healthcheck-failures", "HEALTHCHECK_FAILURES", "Failures before restart"},
	}
	for _, f := range flags {
		if f.env != "" {
			fmt.Fprintf(w, "  %-24s %s (%s)\n", f.name, f.desc, f.env)
		} else {
			fmt.Fprintf(w, "  %-24s %s\n", f.name, f.desc)
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Priority: CLI > ENV > default")
}

// AppArgSlice splits APP_ARGS using strings.Fields.
func (c *Config) AppArgSlice() []string {
	if c == nil || strings.TrimSpace(c.AppArgs) == "" {
		return nil
	}
	return strings.Fields(c.AppArgs)
}

// ResolvedWorkdir returns APP_WORKDIR or the directory containing APP_BINARY.
func (c *Config) ResolvedWorkdir() string {
	if c == nil {
		return ""
	}
	if c.AppWorkdir != "" {
		return c.AppWorkdir
	}
	if c.AppBinary == "" {
		return ""
	}
	return dirOf(c.AppBinary)
}
