// Package config loads and validates QoLauncher configuration from environment
// variables and CLI flags.
package config

import "time"

// RestartPolicy defines when the supervisor restarts the child process.
type RestartPolicy string

const (
	RestartNever     RestartPolicy = "never"
	RestartOnFailure RestartPolicy = "on-failure"
	RestartAlways    RestartPolicy = "always"
)

// Config holds the effective QoLauncher configuration.
type Config struct {
	AppBinary  string
	AppArgs    string
	AppPort    int
	AppWorkdir string

	RestartPolicy   RestartPolicy
	RestartDelay    time.Duration
	MaxRestart      int
	RestartWindow   time.Duration
	RestartBurst    int
	ShutdownTimeout time.Duration

	LogDir           string
	LogRetentionDays int
	LogLevel         string
	TZ               string

	LogPort       int
	LogUsername   string
	LogPassword   string
	ViewerEnabled bool

	HealthcheckEnabled  bool
	HealthcheckType     string
	HealthcheckURL      string
	HealthcheckInterval time.Duration
	HealthcheckTimeout  time.Duration
	HealthcheckFailures int
}

// Default returns a new Config populated with documented defaults.
func Default() *Config {
	return &Config{
		AppArgs:             "",
		AppPort:             8080,
		RestartPolicy:       RestartNever,
		RestartDelay:        3 * time.Second,
		MaxRestart:          0,
		RestartWindow:       60 * time.Second,
		RestartBurst:        10,
		ShutdownTimeout:     30 * time.Second,
		LogDir:              "/var/log/qolauncher",
		LogRetentionDays:    14,
		LogLevel:            "info",
		TZ:                  "UTC",
		LogPort:             8081,
		ViewerEnabled:       true,
		HealthcheckEnabled:  false,
		HealthcheckType:     "http",
		HealthcheckInterval: 30 * time.Second,
		HealthcheckTimeout:  5 * time.Second,
		HealthcheckFailures: 3,
	}
}

// Clone returns a deep copy of the configuration.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	dup := *c
	return &dup
}
