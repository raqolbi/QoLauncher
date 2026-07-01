package viewer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

var logFileNamePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\.log$`)

// LogEntry describes a log file in LOG_DIR.
type LogEntry struct {
	Name       string    `json:"name"`
	Date       string    `json:"date"`
	SizeBytes  int64     `json:"size_bytes"`
	ModifiedAt time.Time `json:"modified_at"`
}

// ListLogs returns daily log files sorted newest first.
func ListLogs(dir string) ([]LogEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	logs := make([]LogEntry, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !logFileNamePattern.MatchString(name) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		date := name[:len("2006-01-02")]
		logs = append(logs, LogEntry{
			Name:       name,
			Date:       date,
			SizeBytes:  info.Size(),
			ModifiedAt: info.ModTime().UTC(),
		})
	}

	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Date > logs[j].Date
	})

	return logs, nil
}

// ValidateFilename ensures filename is a safe daily log name.
func ValidateFilename(name string) error {
	if name == "" || name != filepath.Base(name) {
		return fmt.Errorf("invalid filename")
	}
	if !logFileNamePattern.MatchString(name) {
		return fmt.Errorf("invalid filename")
	}
	return nil
}

// LogFilePath returns the absolute path for a validated log filename.
func LogFilePath(dir, name string) (string, error) {
	if err := ValidateFilename(name); err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}
