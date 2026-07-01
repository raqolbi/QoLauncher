package logger

import "time"

// Supervisor logs documented lifecycle events for the process supervisor.
type Supervisor struct {
	log *Logger
}

// NewSupervisor creates supervisor event helpers backed by l.
func NewSupervisor(l *Logger) *Supervisor {
	return &Supervisor{log: l}
}

func (s *Supervisor) LauncherStarted(version string, fields map[string]any) {
	s.log.Info("launcher started", merge(fields, map[string]any{"version": version}))
}

func (s *Supervisor) ApplicationStarted(binary string, pid, restartCount int) {
	s.log.Info("application started", map[string]any{
		"binary":        binary,
		"pid":           pid,
		"restart_count": restartCount,
	})
}

func (s *Supervisor) ApplicationStopped(pid int, reason string) {
	s.log.Info("application stopped", map[string]any{"pid": pid, "reason": reason})
}

func (s *Supervisor) ApplicationExited(pid, exitCode int, signal string) {
	fields := map[string]any{"pid": pid, "exit_code": exitCode}
	if signal != "" {
		fields["signal"] = signal
	}
	s.log.Info("application exited", fields)
}

func (s *Supervisor) ApplicationCrashed(pid, exitCode int, signal string) {
	fields := map[string]any{"pid": pid, "exit_code": exitCode}
	if signal != "" {
		fields["signal"] = signal
	}
	s.log.Warn("application crashed", fields)
}

func (s *Supervisor) RestartingApplication(delay time.Duration, restartCount int, policy string) {
	s.log.Info("restarting application", map[string]any{
		"delay":         delay.String(),
		"restart_count": restartCount,
		"policy":        policy,
	})
}

func (s *Supervisor) RestartSkipped(reason string, exitCode int, policy string) {
	s.log.Info("restart skipped", map[string]any{
		"reason":    reason,
		"exit_code": exitCode,
		"policy":    policy,
	})
}

func (s *Supervisor) MaximumRestartReached(restartCount, limit int) {
	s.log.Error("maximum restart reached", map[string]any{
		"restart_count": restartCount,
		"limit":         limit,
	})
}

func (s *Supervisor) CrashLoopDetected(restartsInWindow int, window time.Duration) {
	s.log.Error("crash loop detected", map[string]any{
		"restarts_in_window": restartsInWindow,
		"window":             window.String(),
	})
}

func (s *Supervisor) SignalReceived(signal string) {
	s.log.Info("signal received", map[string]any{"signal": signal})
}

func (s *Supervisor) GracefulShutdownCompleted(pid int, duration time.Duration) {
	s.log.Info("graceful shutdown completed", map[string]any{
		"pid":      pid,
		"duration": duration.String(),
	})
}

func (s *Supervisor) ForcedShutdown(pid int, timeout time.Duration) {
	s.log.Warn("forced shutdown", map[string]any{
		"pid":     pid,
		"timeout": timeout.String(),
	})
}

func (s *Supervisor) HealthCheckFailed(url string, consecutiveFailures int) {
	s.log.Warn("health check failed", map[string]any{
		"url":                  url,
		"consecutive_failures": consecutiveFailures,
	})
}

func merge(base, extra map[string]any) map[string]any {
	out := cloneFields(base)
	for k, v := range extra {
		out[k] = v
	}
	return out
}
