package config

import (
	"fmt"
	"os"
)

// Load builds configuration with priority CLI > ENV > default.
func Load(args []string) (*Config, Options, error) {
	cfg := Default()
	if err := ApplyEnvFromOS(cfg); err != nil {
		return nil, Options{}, err
	}

	opts, err := ParseFlags(args, cfg)
	if err != nil {
		return nil, Options{}, err
	}

	if opts.Help() || opts.Version() {
		return cfg, opts, nil
	}

	if err := cfg.Validate(); err != nil {
		return cfg, opts, err
	}

	return cfg, opts, nil
}

// MustLoad loads configuration or exits the process with code 1.
func MustLoad(args []string) (*Config, Options) {
	cfg, opts, err := Load(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	return cfg, opts
}

// ChildEnv returns a copy of the process environment suitable for the child process,
// excluding LAUNCHER_ prefixed variables.
func ChildEnv(environ []string) []string {
	out := make([]string, 0, len(environ))
	for _, entry := range environ {
		key, _, ok := cutEnv(entry)
		if !ok {
			continue
		}
		if len(key) >= 9 && key[:9] == "LAUNCHER_" {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func cutEnv(entry string) (string, string, bool) {
	for i := 0; i < len(entry); i++ {
		if entry[i] == '=' {
			return entry[:i], entry[i+1:], true
		}
	}
	return "", "", false
}
