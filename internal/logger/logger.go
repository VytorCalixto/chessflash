package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level represents the severity of a log message.
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a string into a Level.
func ParseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// Logger is a structured logger with level support.
type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	level    Level
	prefix   string
	fields   map[string]any
	colorize bool
}

// Option configures a Logger.
type Option func(*Logger)

// WithOutput sets the output destination.
func WithOutput(w io.Writer) Option {
	return func(l *Logger) {
		l.out = w
	}
}

// WithLevel sets the minimum log level.
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.level = level
	}
}

// WithPrefix sets a prefix for log messages.
func WithPrefix(prefix string) Option {
	return func(l *Logger) {
		l.prefix = prefix
	}
}

// WithColors enables or disables colorized output.
func WithColors(enabled bool) Option {
	return func(l *Logger) {
		l.colorize = enabled
	}
}

// New creates a new Logger with the given options.
func New(opts ...Option) *Logger {
	l := &Logger{
		out:      os.Stdout,
		level:    INFO,
		fields:   make(map[string]any),
		colorize: true,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Default returns a default logger.
var defaultLogger = New()

// SetDefault sets the default logger.
func SetDefault(l *Logger) {
	defaultLogger = l
}

// Default returns the default logger.
func Default() *Logger {
	return defaultLogger
}

// WithField returns a new logger with the given field added.
func (l *Logger) WithField(key string, value any) *Logger {
	newFields := make(map[string]any, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value
	return &Logger{
		out:      l.out,
		level:    l.level,
		prefix:   l.prefix,
		fields:   newFields,
		colorize: l.colorize,
	}
}

// WithFields returns a new logger with the given fields added.
func (l *Logger) WithFields(fields map[string]any) *Logger {
	newFields := make(map[string]any, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	return &Logger{
		out:      l.out,
		level:    l.level,
		prefix:   l.prefix,
		fields:   newFields,
		colorize: l.colorize,
	}
}

// WithPrefix returns a new logger with the given prefix.
func (l *Logger) WithPrefix(prefix string) *Logger {
	return &Logger{
		out:      l.out,
		level:    l.level,
		prefix:   prefix,
		fields:   l.fields,
		colorize: l.colorize,
	}
}

func (l *Logger) log(level Level, msg string, args ...any) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	var levelStr string
	if l.colorize {
		levelStr = colorize(level)
	} else {
		levelStr = fmt.Sprintf("%-5s", level.String())
	}

	// Get caller info
	_, file, line, ok := runtime.Caller(2)
	caller := ""
	if ok {
		// Extract just the filename
		if idx := strings.LastIndex(file, "/"); idx >= 0 {
			file = file[idx+1:]
		}
		caller = fmt.Sprintf("%s:%d", file, line)
	}

	// Format message
	formattedMsg := msg
	if len(args) > 0 {
		formattedMsg = fmt.Sprintf(msg, args...)
	}

	// Build log line
	var sb strings.Builder
	sb.WriteString(timestamp)
	sb.WriteString(" ")
	sb.WriteString(levelStr)
	sb.WriteString(" ")

	if l.prefix != "" {
		sb.WriteString("[")
		sb.WriteString(l.prefix)
		sb.WriteString("] ")
	}

	if caller != "" {
		sb.WriteString("[")
		sb.WriteString(caller)
		sb.WriteString("] ")
	}

	sb.WriteString(formattedMsg)

	// Add fields
	if len(l.fields) > 0 {
		sb.WriteString(" ")
		first := true
		for k, v := range l.fields {
			if !first {
				sb.WriteString(" ")
			}
			first = false
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(fmt.Sprintf("%v", v))
		}
	}

	sb.WriteString("\n")
	fmt.Fprint(l.out, sb.String())
}

func colorize(level Level) string {
	var color string
	switch level {
	case DEBUG:
		color = "\033[36m" // Cyan
	case INFO:
		color = "\033[32m" // Green
	case WARN:
		color = "\033[33m" // Yellow
	case ERROR:
		color = "\033[31m" // Red
	default:
		color = "\033[0m"
	}
	return fmt.Sprintf("%s%-5s\033[0m", color, level.String())
}

// Debug logs a message at DEBUG level.
func (l *Logger) Debug(msg string, args ...any) {
	l.log(DEBUG, msg, args...)
}

// Info logs a message at INFO level.
func (l *Logger) Info(msg string, args ...any) {
	l.log(INFO, msg, args...)
}

// Warn logs a message at WARN level.
func (l *Logger) Warn(msg string, args ...any) {
	l.log(WARN, msg, args...)
}

// Error logs a message at ERROR level.
func (l *Logger) Error(msg string, args ...any) {
	l.log(ERROR, msg, args...)
}

// Package-level functions that use the default logger.

func Debug(msg string, args ...any) { defaultLogger.Debug(msg, args...) }
func Info(msg string, args ...any)  { defaultLogger.Info(msg, args...) }
func Warn(msg string, args ...any)  { defaultLogger.Warn(msg, args...) }
func Error(msg string, args ...any) { defaultLogger.Error(msg, args...) }

// Context key for request-scoped logger.
type ctxKey struct{}

// FromContext returns the logger from the context, or the default logger.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		return l
	}
	return defaultLogger
}

// NewContext returns a new context with the given logger.
func NewContext(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

