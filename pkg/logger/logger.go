// Package logger provides a simple logging interface for the application.
package logger

import (
	"fmt"
	"os"
	"time"
)

// Level represents a log level type.
type Level = string

const (
	// DEBUG is the log level for debugging messages.
	DEBUG Level = "DEBUG"
	// WARN is the log level for warning messages.
	WARN Level = "WARN"
	// INFO is the log level for informational messages.
	INFO Level = "INFO"
	// ERROR is the log level for error messages.
	ERROR Level = "ERROR"
)

// levelColors maps each log level to a corresponding terminal color for better visibility.
var levelColors = map[Level]string{
	DEBUG: "\033[36m", // Cyan
	WARN:  "\033[33m", // Yellow
	INFO:  "\033[32m", // Green
	ERROR: "\033[31m", // Red
}

// resetColor resets the terminal color to default.
const resetColor = "\033[0m"

// Logger is a simple logger that logs messages at different levels (DEBUG, WARN, INFO, ERROR).
// It supports formatted output with timestamps and colored log levels.
type Logger struct {
	level Level         // The current log level. Logs below this level will be ignored.
	order map[Level]int // Order in which log levels are considered (lower number means higher priority).
}

// New creates and returns a new Logger instance with a given log level.
func New(level Level) *Logger {
	return &Logger{
		level: level,
		order: map[Level]int{
			DEBUG: 0,
			WARN:  1,
			INFO:  2,
			ERROR: 3,
		},
	}
}

// log is a helper function that logs a message with a specific level. It formats the message
// with a timestamp and colored log level, then writes it to stdout.
func (l *Logger) log(lvl Level, format string, args ...any) {
	// Skip logging if the current log level is higher than the desired level.
	if l.order[lvl] < l.order[l.level] {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	coloredLevel := fmt.Sprintf("%s%s%s", levelColors[lvl], lvl, resetColor)
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "[%s] [%s] %s\n", timestamp, coloredLevel, msg)
}

// Debug logs a message with the DEBUG level.
func (l *Logger) Debug(format string, args ...any) {
	l.log(DEBUG, format, args...)
}

// Warn logs a message with the WARN level.
func (l *Logger) Warn(format string, args ...any) {
	l.log(WARN, format, args...)
}

// Info logs a message with the INFO level.
func (l *Logger) Info(format string, args ...any) {
	l.log(INFO, format, args...)
}

// Error logs a message with the ERROR level. It is the highest priority log level.
func (l *Logger) Error(format string, args ...any) {
	l.log(ERROR, format, args...)
}
