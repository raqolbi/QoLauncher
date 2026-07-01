package supervisor

import (
	"context"
	"net/http"
	"time"

	"github.com/raqolbi/qolauncher/internal/config"
	"github.com/raqolbi/qolauncher/internal/logger"
)

type healthChecker struct {
	cfg     *config.Config
	events  *logger.Supervisor
	failCh  chan struct{}
	client  *http.Client
	started bool
}

func newHealthChecker(cfg *config.Config, events *logger.Supervisor) *healthChecker {
	if cfg == nil || !cfg.HealthcheckEnabled {
		return nil
	}
	return &healthChecker{
		cfg:    cfg,
		events: events,
		failCh: make(chan struct{}, 1),
		client: &http.Client{Timeout: cfg.HealthcheckTimeout},
	}
}

func (h *healthChecker) Start(ctx context.Context) {
	if h == nil || h.started {
		return
	}
	h.started = true

	go func() {
		ticker := time.NewTicker(h.cfg.HealthcheckInterval)
		defer ticker.Stop()

		failures := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ok := h.probe()
				if ok {
					failures = 0
					continue
				}
				failures++
				if failures >= h.cfg.HealthcheckFailures {
					h.events.HealthCheckFailed(h.cfg.HealthcheckURL, failures)
					select {
					case h.failCh <- struct{}{}:
					default:
					}
					return
				}
			}
		}
	}()
}

func (h *healthChecker) probe() bool {
	req, err := http.NewRequest(http.MethodGet, h.cfg.HealthcheckURL, nil)
	if err != nil {
		return false
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (h *healthChecker) FailCh() <-chan struct{} {
	if h == nil {
		return nil
	}
	return h.failCh
}
