package rotator

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var logFilePattern = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\.log$`)

// Sweep deletes log files older than retentionDays based on filename date in tz.
// retentionDays <= 0 skips deletion.
func Sweep(dir string, retentionDays int, tz string) error {
	if retentionDays <= 0 {
		return nil
	}

	loc, err := loadLocation(tz)
	if err != nil {
		return err
	}

	cutoff := startOfDay(time.Now().In(loc)).AddDate(0, 0, -retentionDays)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := logFilePattern.FindStringSubmatch(entry.Name())
		if len(matches) != 2 {
			continue
		}
		fileDate, err := time.ParseInLocation("2006-01-02", matches[1], loc)
		if err != nil {
			continue
		}
		if fileDate.Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
	return nil
}

// StartPeriodic runs Sweep on interval until ctx is cancelled.
func StartPeriodic(ctx context.Context, interval time.Duration, dir string, retentionDays int, tz string) {
	if interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = Sweep(dir, retentionDays, tz)
			}
		}
	}()
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func loadLocation(tz string) (*time.Location, error) {
	if tz == "" || tz == "UTC" {
		return time.UTC, nil
	}
	return time.LoadLocation(tz)
}
