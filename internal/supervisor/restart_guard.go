package supervisor

import (
	"errors"
	"time"
)

var (
	errCrashLoop  = errors.New("crash loop detected")
	errMaxRestart = errors.New("maximum restart reached")
)

type restartGuard struct {
	restartCount int
	restartTimes []time.Time
}

func (g *restartGuard) beforeRestart(burst int, window time.Duration, maxRestart int) error {
	now := time.Now()
	g.restartTimes = append(g.restartTimes, now)

	windowStart := now.Add(-window)
	count := 0
	filtered := g.restartTimes[:0]
	for _, ts := range g.restartTimes {
		if !ts.Before(windowStart) {
			count++
			filtered = append(filtered, ts)
		}
	}
	g.restartTimes = filtered

	if count >= burst {
		return errCrashLoop
	}
	if maxRestart > 0 && g.restartCount >= maxRestart {
		return errMaxRestart
	}

	g.restartCount++
	return nil
}

func (g *restartGuard) count() int {
	return g.restartCount
}
