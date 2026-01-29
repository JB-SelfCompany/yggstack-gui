package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.Logger with convenience methods
type Logger struct {
	*zap.SugaredLogger
	underlying *zap.Logger
	ipcCore    *IPCCore // IPC core for sending logs to frontend
}

// Config holds logger configuration
type Config struct {
	Level      string // "debug", "info", "warn", "error"
	FilePath   string // Path to log file (optional)
	Console    bool   // Log to console
	Production bool   // Use production config
}

// DefaultConfig returns default logger configuration
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		Console:    true,
		Production: false,
	}
}

// New creates a new logger with default configuration
func New() *Logger {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new logger with the specified configuration
func NewWithConfig(cfg Config) *Logger {
	var cores []zapcore.Core

	// Parse log level
	level := parseLevel(cfg.Level)

	// Encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if cfg.Production {
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	// Console output
	if cfg.Console {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		consoleCore := zapcore.NewCore(
			consoleEncoder,
			zapcore.AddSync(os.Stdout),
			level,
		)
		cores = append(cores, consoleCore)
	}

	// File output
	if cfg.FilePath != "" {
		// Ensure directory exists
		dir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(dir, 0755); err == nil {
			file, err := os.OpenFile(cfg.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				fileEncoderConfig := encoderConfig
				fileEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
				fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)
				fileCore := zapcore.NewCore(
					fileEncoder,
					zapcore.AddSync(file),
					level,
				)
				cores = append(cores, fileCore)
			}
		}
	}

	// Combine cores
	core := zapcore.NewTee(cores...)

	// Build logger
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &Logger{
		SugaredLogger: zapLogger.Sugar(),
		underlying:    zapLogger,
	}
}

// parseLevel converts a string level to zapcore.Level
func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// SetLevel changes the log level
func (l *Logger) SetLevel(level string) {
	// This would require atomic level in production
	// For now, create a new logger with the new level
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() {
	_ = l.underlying.Sync()
}

// Named creates a named child logger
func (l *Logger) Named(name string) *Logger {
	return &Logger{
		SugaredLogger: l.underlying.Named(name).Sugar(),
		underlying:    l.underlying.Named(name),
	}
}

// With creates a child logger with additional fields
func (l *Logger) With(args ...interface{}) *Logger {
	return &Logger{
		SugaredLogger: l.SugaredLogger.With(args...),
		underlying:    l.underlying,
	}
}

// Zap returns the underlying zap.Logger
func (l *Logger) Zap() *zap.Logger {
	return l.underlying
}

// SetIPCEmitter sets the IPC log emitter for sending logs to frontend
// This should be called after the IPC bridge is initialized
func (l *Logger) SetIPCEmitter(emitter LogEmitter) {
	if l.ipcCore == nil {
		// Create IPC core with current level
		l.ipcCore = NewIPCCore(zapcore.InfoLevel, emitter)

		// Create new tee core with IPC
		newCore := zapcore.NewTee(l.underlying.Core(), l.ipcCore)

		// Replace the underlying logger with the new core
		newLogger := zap.New(newCore, zap.AddCaller(), zap.AddCallerSkip(1))
		l.underlying = newLogger
		l.SugaredLogger = newLogger.Sugar()
	} else {
		// Just update the emitter
		l.ipcCore.SetEmitter(emitter)
	}
}

// SetIPCLogLevel sets the log level for IPC core
func (l *Logger) SetIPCLogLevel(level string) {
	if l.ipcCore != nil {
		l.ipcCore.SetLevel(parseLevel(level))
	}
}

// EnableIPCLogs enables or disables IPC logging
func (l *Logger) EnableIPCLogs(enabled bool) {
	if l.ipcCore != nil {
		l.ipcCore.SetEnabled(enabled)
	}
}

// GetLogs returns logs from the buffer, optionally since a specific timestamp
func (l *Logger) GetLogs(sinceTimestamp int64, limit int) []LogEntry {
	if l.ipcCore != nil {
		return l.ipcCore.GetLogs(sinceTimestamp, limit)
	}
	return nil
}

// GetAllLogs returns all logs from the buffer
func (l *Logger) GetAllLogs() []LogEntry {
	if l.ipcCore != nil {
		return l.ipcCore.GetAllLogs()
	}
	return nil
}

// ClearLogBuffer clears the log buffer
func (l *Logger) ClearLogBuffer() {
	if l.ipcCore != nil {
		l.ipcCore.ClearBuffer()
	}
}

// IPCCore returns the IPC core for direct access (used by handlers)
func (l *Logger) IPCCore() *IPCCore {
	return l.ipcCore
}

// Writer returns an io.Writer that writes to the logger
// Used for compatibility with libraries that need io.Writer
func (l *Logger) Writer() *logWriter {
	return &logWriter{logger: l}
}

// logWriter wraps Logger to implement io.Writer
type logWriter struct {
	logger *Logger
}

// Write implements io.Writer
func (w *logWriter) Write(p []byte) (n int, err error) {
	// Remove trailing newline if present
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}
	w.logger.Info(msg)
	return len(p), nil
}

// GetLogFilePath returns the default log file path
// PORTABLE MODE: logs are stored in "data/logs" subdirectory next to the executable
func GetLogFilePath() string {
	// Get executable directory for portable mode
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to current working directory
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, "data", "logs", "yggstack-gui.log")
	}
	// Resolve symlinks to get the real path
	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return filepath.Join(filepath.Dir(exePath), "data", "logs", "yggstack-gui.log")
	}
	return filepath.Join(filepath.Dir(realPath), "data", "logs", "yggstack-gui.log")
}
