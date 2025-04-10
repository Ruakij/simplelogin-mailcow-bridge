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

// Reset color code
const colorReset = "\033[0m"

// Logger wraps the standard logger with levels
type Logger struct {
	level      int
	useColors  bool
	stdLogger  *log.Logger
	mu         sync.Mutex
	component  string
	requestID  string
	sessionID  string
	useConsole bool
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// initialize the default logger
func init() {
	// Set up default logger with INFO level
	defaultLogger = newLogger(os.Stdout, LevelInfo, true, "")
}

// newLogger creates a new logger instance
func newLogger(out io.Writer, level int, useColors bool, component string) *Logger {
	return &Logger{
		level:      level,
		useColors:  useColors,
		stdLogger:  log.New(out, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC|log.Lshortfile),
		component:  component,
		useConsole: true,
	}
}

// parseLevel converts a string level to its corresponding constant
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
		return LevelInfo // Default to INFO
	}
}

// WithComponent creates a new logger with a specific component name
func (l *Logger) WithComponent(component string) *Logger {
	newLogger := &Logger{
		level:      l.level,
		useColors:  l.useColors,
		stdLogger:  l.stdLogger,
		component:  component,
		requestID:  l.requestID,
		sessionID:  l.sessionID,
		useConsole: l.useConsole,
	}
	return newLogger
}

// WithRequestID creates a new logger with a request ID
func (l *Logger) WithRequestID(id string) *Logger {
	newLogger := &Logger{
		level:      l.level,
		useColors:  l.useColors,
		stdLogger:  l.stdLogger,
		component:  l.component,
		requestID:  id,
		sessionID:  l.sessionID,
		useConsole: l.useConsole,
	}
	return newLogger
}

// WithSessionID creates a new logger with a session ID
func (l *Logger) WithSessionID(id string) *Logger {
	newLogger := &Logger{
		level:      l.level,
		useColors:  l.useColors,
		stdLogger:  l.stdLogger,
		component:  l.component,
		requestID:  l.requestID,
		sessionID:  id,
		useConsole: l.useConsole,
	}
	return newLogger
}

// formatPrefix creates a log prefix with level, component, and context info
func (l *Logger) formatPrefix(level int) string {
	var contextInfo strings.Builder

	if l.requestID != "" {
		contextInfo.WriteString(fmt.Sprintf("[%s] ", l.requestID))
	}
	if l.sessionID != "" {
		contextInfo.WriteString(fmt.Sprintf("(session:%s) ", l.sessionID))
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

// log logs a message at the specified level
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

	if l.useConsole {
		l.stdLogger.Output(3, prefix+msg) // 3 for correct file position
	}
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

// GetTimestamp returns a formatted timestamp for logging
func GetTimestamp() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05.000")
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
