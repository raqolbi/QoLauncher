package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Level is a launcher log severity.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger writes structured logs to an io.Writer (typically stderr).
type Logger struct {
	level Level
	out   io.Writer
}

// New creates a Logger with the given level name and output writer.
func New(level string, out io.Writer) *Logger {
	if out == nil {
		out = os.Stderr
	}
	return &Logger{level: ParseLevel(level), out: out}
}

// ParseLevel converts a level string to Level. Unknown values default to info.
func ParseLevel(level string) Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func (l *Logger) log(level Level, name, msg string, fields map[string]any) {
	if l == nil {
		return
	}
	if level < l.level {
		return
	}

	ts := time.Now().UTC().Format(time.RFC3339)
	var b strings.Builder
	fmt.Fprintf(&b, "%s level=%s msg=%q", ts, name, msg)
	for _, key := range sortedKeys(fields) {
		fmt.Fprintf(&b, " %s=%s", key, formatValue(fields[key]))
	}
	b.WriteByte('\n')
	_, _ = io.WriteString(l.out, b.String())
}

func (l *Logger) Debug(msg string, fields map[string]any) { l.log(LevelDebug, "debug", msg, fields) }
func (l *Logger) Info(msg string, fields map[string]any)  { l.log(LevelInfo, "info", msg, fields) }
func (l *Logger) Warn(msg string, fields map[string]any)  { l.log(LevelWarn, "warn", msg, fields) }
func (l *Logger) Error(msg string, fields map[string]any) { l.log(LevelError, "error", msg, fields) }

// With returns a FieldLogger that attaches constant fields to each event.
func (l *Logger) With(fields map[string]any) *FieldLogger {
	return &FieldLogger{logger: l, fields: cloneFields(fields)}
}

// FieldLogger logs events with preset fields.
type FieldLogger struct {
	logger *Logger
	fields map[string]any
}

func (f *FieldLogger) merge(extra map[string]any) map[string]any {
	out := cloneFields(f.fields)
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func (f *FieldLogger) Debug(msg string, extra map[string]any) {
	f.logger.Debug(msg, f.merge(extra))
}
func (f *FieldLogger) Info(msg string, extra map[string]any)  { f.logger.Info(msg, f.merge(extra)) }
func (f *FieldLogger) Warn(msg string, extra map[string]any)  { f.logger.Warn(msg, f.merge(extra)) }
func (f *FieldLogger) Error(msg string, extra map[string]any) { f.logger.Error(msg, f.merge(extra)) }

func cloneFields(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func formatValue(v any) string {
	switch x := v.(type) {
	case string:
		if strings.ContainsAny(x, " \t") {
			return fmt.Sprintf("%q", x)
		}
		return x
	case fmt.Stringer:
		return formatValue(x.String())
	default:
		return fmt.Sprintf("%v", x)
	}
}
