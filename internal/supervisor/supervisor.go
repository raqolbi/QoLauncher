package supervisor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/raqolbi/qolauncher/internal/capture"
	"github.com/raqolbi/qolauncher/internal/config"
	"github.com/raqolbi/qolauncher/internal/logger"
	"github.com/raqolbi/qolauncher/internal/logwriter"
)

// Supervisor manages the child application lifecycle.
type Supervisor struct {
	cfg    *config.Config
	events *logger.Supervisor
	writer *logwriter.Writer

	guard restartGuard

	shuttingDown bool
	healthStop   bool
}

// New creates a Supervisor.
func New(cfg *config.Config, events *logger.Supervisor, writer *logwriter.Writer) *Supervisor {
	return &Supervisor{cfg: cfg, events: events, writer: writer}
}

type childProcess struct {
	cmd  *exec.Cmd
	done chan ExitResult
	pid  int
}

// Run supervises the configured application until final exit.
func (s *Supervisor) Run(ctx context.Context) int {
	sigCh := make(chan os.Signal, 1)
	registerSignals(sigCh)

	health := newHealthChecker(s.cfg, s.events)
	healthCtx, healthCancel := context.WithCancel(ctx)
	defer healthCancel()
	health.Start(healthCtx)

	for {
		child, err := s.startChild(ctx)
		if err != nil {
			s.events.ApplicationStartFailed(s.cfg.AppBinary, err)
			return 1
		}

		exitCode := s.waitCycle(sigCh, health, child)
		if exitCode >= 0 {
			return exitCode
		}

		if err := s.prepareRestart(); err != nil {
			if errors.Is(err, errCrashLoop) {
				s.events.CrashLoopDetected(s.cfg.RestartBurst, s.cfg.RestartWindow)
				return 1
			}
			if errors.Is(err, errMaxRestart) {
				s.events.MaximumRestartReached(s.guard.count(), s.cfg.MaxRestart)
				return 1
			}
			return 1
		}

		s.events.RestartingApplication(
			s.cfg.RestartDelay,
			s.guard.count(),
			string(s.cfg.RestartPolicy),
		)
		if s.cfg.RestartDelay > 0 {
			select {
			case <-ctx.Done():
				return 0
			case <-time.After(s.cfg.RestartDelay):
			}
		}
	}
}

func (s *Supervisor) waitCycle(sigCh <-chan os.Signal, health *healthChecker, child *childProcess) int {
	var healthCh <-chan struct{}
	if health != nil {
		healthCh = health.FailCh()
	}

	for {
		select {
		case sig := <-sigCh:
			return s.handleSignal(child, sig)
		case <-healthCh:
			s.healthStop = true
			res := s.terminateChild(child)
			return s.handleExit(child.pid, res)
		case res := <-child.done:
			return s.handleExit(child.pid, res)
		}
	}
}

func (s *Supervisor) startChild(ctx context.Context) (*childProcess, error) {
	cmd := exec.CommandContext(ctx, s.cfg.AppBinary, s.cfg.AppArgSlice()...)
	cmd.Dir = s.cfg.ResolvedWorkdir()
	cmd.Env = config.ChildEnv(os.Environ())

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	s.healthStop = false
	pid := cmd.Process.Pid
	s.events.ApplicationStarted(s.cfg.AppBinary, pid, s.guard.count())

	done := make(chan ExitResult, 1)
	var wg sync.WaitGroup
	capture.Pipes(stdout, stderr, s.writer, &wg)

	go func() {
		waitErr := cmd.Wait()
		res := ParseExit(waitErr)
		wg.Wait()
		done <- res
	}()

	return &childProcess{cmd: cmd, done: done, pid: pid}, nil
}

func (s *Supervisor) handleExit(pid int, res ExitResult) int {
	if s.shuttingDown {
		if res.Failed() {
			s.events.ForcedShutdown(pid, s.cfg.ShutdownTimeout)
			return 137
		}
		s.events.GracefulShutdownCompleted(pid, 0)
		return 0
	}

	if !res.Failed() {
		s.events.ApplicationExited(pid, res.Code, res.SignalName())
	} else {
		s.events.ApplicationCrashed(pid, res.Code, res.SignalName())
	}

	restart, exitCode := s.shouldRestart(res)
	if !restart {
		s.events.RestartSkipped("policy", res.Code, string(s.cfg.RestartPolicy))
		return exitCode
	}
	return -1
}

func (s *Supervisor) shouldRestart(res ExitResult) (bool, int) {
	if s.shuttingDown {
		return false, 0
	}

	failed := res.Failed() || s.healthStop

	switch s.cfg.RestartPolicy {
	case config.RestartNever:
		return false, finalExitCode(res)
	case config.RestartOnFailure:
		if failed {
			return true, -1
		}
		return false, 0
	case config.RestartAlways:
		return true, -1
	default:
		return false, finalExitCode(res)
	}
}

func finalExitCode(res ExitResult) int {
	if res.Code != 0 {
		return res.Code
	}
	if res.Signal != 0 {
		return 128 + int(res.Signal)
	}
	return 0
}

func (s *Supervisor) prepareRestart() error {
	return s.guard.beforeRestart(s.cfg.RestartBurst, s.cfg.RestartWindow, s.cfg.MaxRestart)
}

func (s *Supervisor) handleSignal(child *childProcess, sig os.Signal) int {
	s.shuttingDown = true
	s.events.SignalReceived(sig.String())

	if child.cmd.Process != nil {
		_ = child.cmd.Process.Signal(sig)
	}

	start := time.Now()
	timer := time.NewTimer(s.cfg.ShutdownTimeout)
	defer timer.Stop()

	select {
	case res := <-child.done:
		if res.Failed() {
			s.events.ForcedShutdown(child.pid, s.cfg.ShutdownTimeout)
			return 137
		}
		s.events.GracefulShutdownCompleted(child.pid, time.Since(start))
		return 0
	case <-timer.C:
		if child.cmd.Process != nil {
			_ = child.cmd.Process.Kill()
		}
		res := <-child.done
		s.events.ForcedShutdown(child.pid, s.cfg.ShutdownTimeout)
		if res.Failed() {
			return 137
		}
		return 0
	}
}

func (s *Supervisor) terminateChild(child *childProcess) ExitResult {
	if child.cmd.Process != nil {
		_ = child.cmd.Process.Signal(syscall.SIGTERM)
	}

	timer := time.NewTimer(s.cfg.ShutdownTimeout)
	defer timer.Stop()

	select {
	case res := <-child.done:
		return res
	case <-timer.C:
		if child.cmd.Process != nil {
			_ = child.cmd.Process.Kill()
		}
		return <-child.done
	}
}
