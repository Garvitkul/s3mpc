package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel parses a string into a LogLevel
func ParseLogLevel(level string) (LogLevel, error) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return LevelDebug, nil
	case "INFO":
		return LevelInfo, nil
	case "WARN", "WARNING":
		return LevelWarn, nil
	case "ERROR":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("invalid log level: %s", level)
	}
}

// Logger represents a structured logger
type Logger struct {
	level  LogLevel
	output io.Writer
	quiet  bool
}

// NewLogger creates a new logger instance
func NewLogger(level LogLevel, output io.Writer, quiet bool) *Logger {
	return &Logger{
		level:  level,
		output: output,
		quiet:  quiet,
	}
}

// NewConsoleLogger creates a logger that writes to stdout/stderr
func NewConsoleLogger(verbose, quiet bool) *Logger {
	level := LevelInfo
	if verbose {
		level = LevelDebug
	}
	if quiet {
		level = LevelError
	}
	
	return NewLogger(level, os.Stderr, quiet)
}

// NewFileLogger creates a logger that writes to a file
func NewFileLogger(filename string, level LogLevel) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", filename, err)
	}
	
	return NewLogger(level, file, false), nil
}

// NewMultiLogger creates a logger that writes to multiple outputs
func NewMultiLogger(loggers ...*Logger) *Logger {
	if len(loggers) == 0 {
		return NewConsoleLogger(false, false)
	}
	
	if len(loggers) == 1 {
		return loggers[0]
	}
	
	// Find the minimum log level
	minLevel := LevelError
	var writers []io.Writer
	quiet := true
	
	for _, logger := range loggers {
		if logger.level < minLevel {
			minLevel = logger.level
		}
		writers = append(writers, logger.output)
		if !logger.quiet {
			quiet = false
		}
	}
	
	multiWriter := io.MultiWriter(writers...)
	return NewLogger(minLevel, multiWriter, quiet)
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// IsEnabled checks if a log level is enabled
func (l *Logger) IsEnabled(level LogLevel) bool {
	return level >= l.level
}

// log writes a log message with the specified level
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if !l.IsEnabled(level) {
		return
	}
	
	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	
	// Build the log message
	var parts []string
	parts = append(parts, fmt.Sprintf("[%s]", timestamp))
	parts = append(parts, fmt.Sprintf("[%s]", level.String()))
	parts = append(parts, message)
	
	// Add fields if any
	if len(fields) > 0 {
		var fieldParts []string
		for key, value := range fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", key, value))
		}
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(fieldParts, ", ")))
	}
	
	logLine := strings.Join(parts, " ") + "\n"
	
	// Write to output
	l.output.Write([]byte(logLine))
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	l.log(LevelDebug, message, fieldMap)
}

// Info logs an info message
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	l.log(LevelInfo, message, fieldMap)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	l.log(LevelWarn, message, fieldMap)
}

// Error logs an error message
func (l *Logger) Error(message string, fields ...map[string]interface{}) {
	var fieldMap map[string]interface{}
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	l.log(LevelError, message, fieldMap)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Debug(fmt.Sprintf(format, args...))
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

// WithFields creates a new logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *FieldLogger {
	return &FieldLogger{
		logger: l,
		fields: fields,
	}
}

// FieldLogger is a logger with predefined fields
type FieldLogger struct {
	logger *Logger
	fields map[string]interface{}
}

// Debug logs a debug message with predefined fields
func (fl *FieldLogger) Debug(message string) {
	fl.logger.log(LevelDebug, message, fl.fields)
}

// Info logs an info message with predefined fields
func (fl *FieldLogger) Info(message string) {
	fl.logger.log(LevelInfo, message, fl.fields)
}

// Warn logs a warning message with predefined fields
func (fl *FieldLogger) Warn(message string) {
	fl.logger.log(LevelWarn, message, fl.fields)
}

// Error logs an error message with predefined fields
func (fl *FieldLogger) Error(message string) {
	fl.logger.log(LevelError, message, fl.fields)
}

// Debugf logs a formatted debug message with predefined fields
func (fl *FieldLogger) Debugf(format string, args ...interface{}) {
	fl.logger.log(LevelDebug, fmt.Sprintf(format, args...), fl.fields)
}

// Infof logs a formatted info message with predefined fields
func (fl *FieldLogger) Infof(format string, args ...interface{}) {
	fl.logger.log(LevelInfo, fmt.Sprintf(format, args...), fl.fields)
}

// Warnf logs a formatted warning message with predefined fields
func (fl *FieldLogger) Warnf(format string, args ...interface{}) {
	fl.logger.log(LevelWarn, fmt.Sprintf(format, args...), fl.fields)
}

// Errorf logs a formatted error message with predefined fields
func (fl *FieldLogger) Errorf(format string, args ...interface{}) {
	fl.logger.log(LevelError, fmt.Sprintf(format, args...), fl.fields)
}

// Global logger instance
var globalLogger *Logger

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		globalLogger = NewConsoleLogger(false, false)
	}
	return globalLogger
}

// Global logging functions that use the global logger

// Debug logs a debug message using the global logger
func Debug(message string, fields ...map[string]interface{}) {
	GetGlobalLogger().Debug(message, fields...)
}

// Info logs an info message using the global logger
func Info(message string, fields ...map[string]interface{}) {
	GetGlobalLogger().Info(message, fields...)
}

// Warn logs a warning message using the global logger
func Warn(message string, fields ...map[string]interface{}) {
	GetGlobalLogger().Warn(message, fields...)
}

// Error logs an error message using the global logger
func Error(message string, fields ...map[string]interface{}) {
	GetGlobalLogger().Error(message, fields...)
}

// Debugf logs a formatted debug message using the global logger
func Debugf(format string, args ...interface{}) {
	GetGlobalLogger().Debugf(format, args...)
}

// Infof logs a formatted info message using the global logger
func Infof(format string, args ...interface{}) {
	GetGlobalLogger().Infof(format, args...)
}

// Warnf logs a formatted warning message using the global logger
func Warnf(format string, args ...interface{}) {
	GetGlobalLogger().Warnf(format, args...)
}

// Errorf logs a formatted error message using the global logger
func Errorf(format string, args ...interface{}) {
	GetGlobalLogger().Errorf(format, args...)
}