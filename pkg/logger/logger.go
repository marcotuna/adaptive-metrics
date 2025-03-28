// Package logger provides flexible logging functionality for the adaptive metrics system
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/marcotuna/adaptive-metrics/internal/config"
)

// Level represents logging levels
type Level int

const (
	// Debug level for highly detailed diagnostic information
	Debug Level = iota
	// Info level for general operational information
	Info
	// Warn level for potentially harmful situations
	Warn
	// Error level for errors that might still allow the application to continue running
	Error
)

// String returns the string representation of a log level
func (l Level) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging capabilities
type Logger struct {
	level         Level
	format        string
	output        io.Writer
	includeTime   bool
	includeCaller bool
}

// Fields represents a collection of log fields
type Fields map[string]interface{}

// Default global logger instance
var (
	defaultLogger *Logger
	once          sync.Once
)

// Init initializes the global logger with the provided configuration
func Init(cfg *config.LoggingConfig) error {
	var err error
	once.Do(func() {
		defaultLogger, err = New(cfg)
	})
	return err
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if defaultLogger == nil {
		// Return a default logger to stdout if not initialized
		defaultLogger = &Logger{
			level:         Info,
			format:        "json",
			output:        os.Stdout,
			includeTime:   true,
			includeCaller: false,
		}
	}
	return defaultLogger
}

// New creates a new logger instance based on the provided configuration
func New(cfg *config.LoggingConfig) (*Logger, error) {
	var output io.Writer = os.Stdout
	var err error

	// Configure output destination
	if cfg.File != "" {
		output, err = os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
	}

	// Configure log level
	level := Info // Default level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = Debug
	case "info":
		level = Info
	case "warn":
		level = Warn
	case "error":
		level = Error
	}

	// Determine format
	format := "json" // Default format
	if cfg.Format != "" {
		format = strings.ToLower(cfg.Format)
	}

	return &Logger{
		level:         level,
		format:        format,
		output:        output,
		includeTime:   cfg.IncludeTimestamp,
		includeCaller: cfg.IncludeCaller,
	}, nil
}

// log logs a message at the specified level with fields
func (l *Logger) log(level Level, msg string, fields Fields) {
	if level < l.level {
		return
	}

	// Merge standard fields
	logFields := Fields{}
	if fields != nil {
		for k, v := range fields {
			logFields[k] = v
		}
	}

	// Add standard metadata
	logFields["level"] = level.String()
	logFields["message"] = msg

	if l.includeTime {
		logFields["timestamp"] = time.Now().Format(time.RFC3339)
	}

	if l.includeCaller {
		_, file, line, ok := getCaller(2) // Skip this function and the calling log function
		if ok {
			logFields["caller"] = fmt.Sprintf("%s:%d", file, line)
		}
	}

	if l.format == "json" {
		l.logJSON(logFields)
	} else {
		l.logText(level, msg, logFields)
	}
}

// logJSON formats and writes a JSON log entry
func (l *Logger) logJSON(fields Fields) {
	jsonData, err := json.Marshal(fields)
	if err != nil {
		log.Printf("Error marshaling log fields to JSON: %v", err)
		return
	}

	fmt.Fprintln(l.output, string(jsonData))
}

// logText formats and writes a text log entry
func (l *Logger) logText(level Level, msg string, fields Fields) {
	var builder strings.Builder

	// Add timestamp if enabled
	if l.includeTime {
		builder.WriteString(time.Now().Format(time.RFC3339))
		builder.WriteString(" ")
	}

	// Add level
	builder.WriteString("[")
	builder.WriteString(level.String())
	builder.WriteString("] ")

	// Add message
	builder.WriteString(msg)

	// Add caller if enabled
	if caller, ok := fields["caller"]; ok && l.includeCaller {
		builder.WriteString(" (")
		builder.WriteString(caller.(string))
		builder.WriteString(")")
	}

	// Add the rest of the fields
	for k, v := range fields {
		if k != "level" && k != "message" && k != "timestamp" && k != "caller" {
			builder.WriteString(" ")
			builder.WriteString(k)
			builder.WriteString("=")
			builder.WriteString(fmt.Sprintf("%v", v))
		}
	}

	fmt.Fprintln(l.output, builder.String())
}

// Debug logs a message at the Debug level
func (l *Logger) Debug(msg string) {
	l.log(Debug, msg, nil)
}

// Debugf logs a formatted message at the Debug level
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(Debug, fmt.Sprintf(format, args...), nil)
}

// DebugWithFields logs a message at the Debug level with additional fields
func (l *Logger) DebugWithFields(msg string, fields Fields) {
	l.log(Debug, msg, fields)
}

// Info logs a message at the Info level
func (l *Logger) Info(msg string) {
	l.log(Info, msg, nil)
}

// Infof logs a formatted message at the Info level
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(Info, fmt.Sprintf(format, args...), nil)
}

// InfoWithFields logs a message at the Info level with additional fields
func (l *Logger) InfoWithFields(msg string, fields Fields) {
	l.log(Info, msg, fields)
}

// Warn logs a message at the Warn level
func (l *Logger) Warn(msg string) {
	l.log(Warn, msg, nil)
}

// Warnf logs a formatted message at the Warn level
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(Warn, fmt.Sprintf(format, args...), nil)
}

// WarnWithFields logs a message at the Warn level with additional fields
func (l *Logger) WarnWithFields(msg string, fields Fields) {
	l.log(Warn, msg, fields)
}

// Error logs a message at the Error level
func (l *Logger) Error(msg string) {
	l.log(Error, msg, nil)
}

// Errorf logs a formatted message at the Error level
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(Error, fmt.Sprintf(format, args...), nil)
}

// ErrorWithFields logs a message at the Error level with additional fields
func (l *Logger) ErrorWithFields(msg string, fields Fields) {
	l.log(Error, msg, fields)
}

// Global convenience functions that use the default logger

// LogDebug logs a message at the Debug level
func LogDebug(msg string) {
	GetLogger().Debug(msg)
}

// LogDebugf logs a formatted message at the Debug level
func LogDebugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

// LogDebugWithFields logs a message at the Debug level with additional fields
func LogDebugWithFields(msg string, fields Fields) {
	GetLogger().DebugWithFields(msg, fields)
}

// LogInfo logs a message at the Info level
func LogInfo(msg string) {
	GetLogger().Info(msg)
}

// LogInfof logs a formatted message at the Info level
func LogInfof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

// LogInfoWithFields logs a message at the Info level with additional fields
func LogInfoWithFields(msg string, fields Fields) {
	GetLogger().InfoWithFields(msg, fields)
}

// LogWarn logs a message at the Warn level
func LogWarn(msg string) {
	GetLogger().Warn(msg)
}

// LogWarnf logs a formatted message at the Warn level
func LogWarnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

// LogWarnWithFields logs a message at the Warn level with additional fields
func LogWarnWithFields(msg string, fields Fields) {
	GetLogger().WarnWithFields(msg, fields)
}

// LogError logs a message at the Error level
func LogError(msg string) {
	GetLogger().Error(msg)
}

// LogErrorf logs a formatted message at the Error level
func LogErrorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// LogErrorWithFields logs a message at the Error level with additional fields
func LogErrorWithFields(msg string, fields Fields) {
	GetLogger().ErrorWithFields(msg, fields)
}

// getCaller returns the filename and line number of the caller
func getCaller(skip int) (pc uintptr, file string, line int, ok bool) {
	return getCaller2(skip + 1)
}

// getCaller2 retrieves the caller information using the runtime package
func getCaller2(skip int) (pc uintptr, file string, line int, ok bool) {
	pc, file, line, ok = runtime.Caller(skip)
	if !ok {
		return 0, "unknown", 0, false
	}

	// Extract just the filename, not the full path
	if index := strings.LastIndex(file, "/"); index >= 0 {
		file = file[index+1:]
	}

	return pc, file, line, ok
}
