package logwriter

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	StreamStdout = "stdout"
	StreamStderr = "stderr"
)

// Writer appends captured application output to daily log files.
type Writer struct {
	dir  string
	loc  *time.Location
	now  func() time.Time

	mu      sync.Mutex
	curDate string
	file    *os.File
}

// New creates a Writer that stores logs under dir using timezone tz (IANA name).
func New(dir, tz string) (*Writer, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	loc, err := loadLocation(tz)
	if err != nil {
		return nil, err
	}

	w := &Writer{
		dir: dir,
		loc: loc,
		now: time.Now,
	}
	if err := w.rotateIfNeededLocked(); err != nil {
		return nil, err
	}
	return w, nil
}

// WriteLine writes a single captured line from stream (stdout/stderr).
func (w *Writer) WriteLine(stream, message string) error {
	if w == nil {
		return fmt.Errorf("log writer is nil")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.rotateIfNeededLocked(); err != nil {
		return err
	}

	ts := w.now().In(w.loc).Format(time.RFC3339)
	line := fmt.Sprintf("%s [%s] %s\n", ts, stream, message)
	_, err := w.file.WriteString(line)
	return err
}

// Close flushes and closes the underlying file.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

// Dir returns the log directory path.
func (w *Writer) Dir() string {
	if w == nil {
		return ""
	}
	return w.dir
}

func (w *Writer) rotateIfNeededLocked() error {
	date := w.now().In(w.loc).Format("2006-01-02")
	if w.file != nil && date == w.curDate {
		return nil
	}
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}

	path := filepath.Join(w.dir, date+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w.file = f
	w.curDate = date
	return nil
}

func loadLocation(tz string) (*time.Location, error) {
	if tz == "" || tz == "UTC" {
		return time.UTC, nil
	}
	return time.LoadLocation(tz)
}

// SetNowFunc overrides the clock source (for tests).
func (w *Writer) SetNowFunc(fn func() time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.now = fn
}
