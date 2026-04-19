package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// Log levels
const (
	LevelDebug = iota
	LevelInfo
	LevelWarn
	LevelError
)

var levelNames = map[int]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

var levelColors = map[int]string{
	LevelDebug: "\033[36m", // Cyan
	LevelInfo:  "\033[32m", // Green
	LevelWarn:  "\033[33m", // Yellow
	LevelError: "\033[31m", // Red
}

const colorReset = "\033[0m"

// Logger wraps the standard logger with levels
type Logger struct {
	level     int
	useColors bool
	stdLogger *log.Logger
	mu        sync.Mutex
	component string
	requestID string
}

var defaultLogger *Logger

func init() {
	defaultLogger = newLogger(os.Stdout, LevelInfo, true, "")
}

func newLogger(out io.Writer, level int, useColors bool, component string) *Logger {
	return &Logger{
		level:     level,
		useColors: useColors,
		stdLogger: log.New(out, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC|log.Lshortfile),
		component: component,
	}
}

func parseLevel(level string) int {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN":
		return LevelWarn
	case "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}

// Mask returns a safely masked representation of data for logs.
//
// If the input contains an email address, only the local part is masked and the
// domain is preserved.
func Mask(data string, percent float64) string {
	if data == "" {
		return ""
	}

	if at := strings.IndexByte(data, '@'); at > 0 {
		local := data[:at]
		domain := data[at:]
		return Mask(local, percent) + domain
	}

	if percent <= 0 {
		percent = 0.5
	}
	if percent > 1 {
		percent = 1
	}

	length := len(data)
	if length <= 2 {
		return strings.Repeat("*", length)
	}

	keep := int(float64(length) * (1 - percent))
	if keep < 1 {
		keep = 1
	}
	if keep > length-1 {
		keep = length - 1
	}

	return data[:keep] + "***"
}

func (l *Logger) clone() *Logger {
	c := *l
	return &c
}

// WithComponent creates a new logger with a specific component name
func (l *Logger) WithComponent(component string) *Logger {
	c := l.clone()
	c.component = component
	return c
}

// WithRequestID creates a new logger with a request ID
func (l *Logger) WithRequestID(id string) *Logger {
	c := l.clone()
	c.requestID = id
	return c
}

// formatPrefix creates a log prefix with level, component, and context info
func (l *Logger) formatPrefix(level int) string {
	var contextInfo strings.Builder

	if l.requestID != "" {
		contextInfo.WriteString(fmt.Sprintf("[%s] ", l.requestID))
	}
	if l.component != "" {
		contextInfo.WriteString(fmt.Sprintf("<%s> ", l.component))
	}

	levelStr := levelNames[level]
	if l.useColors {
		return fmt.Sprintf("%s[%s]%s %s", levelColors[level], levelStr, colorReset, contextInfo.String())
	}
	return fmt.Sprintf("[%s] %s", levelStr, contextInfo.String())
}

func (l *Logger) log(level int, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	prefix := l.formatPrefix(level)
	var msg string
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	} else {
		msg = format
	}

	l.stdLogger.Output(3, prefix+msg)
}

// Debug logs a message at DEBUG level
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs a message at INFO level
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a message at WARN level
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs a message at ERROR level
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Fatal logs an error message, then exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
	os.Exit(1)
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetColor enables or disables colored output
func (l *Logger) SetColor(enable bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.useColors = enable
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() int {
	return l.level
}

// Global functions that use the default logger

// Default returns the default logger
func Default() *Logger {
	return defaultLogger
}

// SetupGlobal configures the global default logger
func SetupGlobal(level string, useColors bool) {
	logLevel := parseLevel(level)
	defaultLogger.SetLevel(logLevel)
	defaultLogger.SetColor(useColors)
}

// WithComponent creates a new logger from the default with a specific component
func WithComponent(component string) *Logger {
	return defaultLogger.WithComponent(component)
}

// WithRequestID creates a new logger from the default with a request ID
func WithRequestID(id string) *Logger {
	return defaultLogger.WithRequestID(id)
}

// Debug logs a message at DEBUG level with the default logger
func Debug(format string, args ...interface{}) {
	defaultLogger.log(LevelDebug, format, args...)
}

// Info logs a message at INFO level with the default logger
func Info(format string, args ...interface{}) {
	defaultLogger.log(LevelInfo, format, args...)
}

// Warn logs a message at WARN level with the default logger
func Warn(format string, args ...interface{}) {
	defaultLogger.log(LevelWarn, format, args...)
}

// Error logs a message at ERROR level with the default logger
func Error(format string, args ...interface{}) {
	defaultLogger.log(LevelError, format, args...)
}

// Fatal logs a message at ERROR level with the default logger, then exits
func Fatal(format string, args ...interface{}) {
	defaultLogger.log(LevelError, format, args...)
	os.Exit(1)
}

// FormatDuration returns a human-readable representation of a duration
func FormatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%d µs", d.Microseconds())
	} else if d < time.Second {
		return fmt.Sprintf("%.2f ms", float64(d.Microseconds())/1000)
	} else if d < time.Minute {
		return fmt.Sprintf("%.2f s", d.Seconds())
	}
	return fmt.Sprintf("%.2f m", d.Minutes())
}
