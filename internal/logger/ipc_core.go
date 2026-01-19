package logger

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogEntry represents a log entry to be sent to frontend
type LogEntry struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// LogEmitter is a function that emits log entries to the frontend
type LogEmitter func(entry LogEntry)

// IPCCore is a zapcore.Core that sends log entries via IPC
type IPCCore struct {
	mu       sync.RWMutex
	level    zapcore.Level
	emitter  LogEmitter
	fields   []zapcore.Field
	name     string
	enabled  bool

	// Log buffer for polling
	buffer    []LogEntry
	bufferMax int
	lastID    int64
}

// NewIPCCore creates a new IPC core for logging
func NewIPCCore(level zapcore.Level, emitter LogEmitter) *IPCCore {
	return &IPCCore{
		level:     level,
		emitter:   emitter,
		fields:    make([]zapcore.Field, 0),
		enabled:   true,
		buffer:    make([]LogEntry, 0, 1000),
		bufferMax: 1000,
	}
}

// Enabled checks if the core is enabled for a given log level
func (c *IPCCore) Enabled(level zapcore.Level) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled && level >= c.level
}

// With creates a new core with additional fields
func (c *IPCCore) With(fields []zapcore.Field) zapcore.Core {
	c.mu.RLock()
	defer c.mu.RUnlock()

	newFields := make([]zapcore.Field, len(c.fields)+len(fields))
	copy(newFields, c.fields)
	copy(newFields[len(c.fields):], fields)

	// Return a new core that shares the buffer with the parent
	return &IPCCore{
		level:     c.level,
		emitter:   c.emitter,
		fields:    newFields,
		name:      c.name,
		enabled:   c.enabled,
		buffer:    c.buffer, // Share buffer reference
		bufferMax: c.bufferMax,
	}
}

// Check determines whether the entry should be logged
func (c *IPCCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

// Write logs an entry
func (c *IPCCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	c.mu.Lock()
	emitter := c.emitter
	enabled := c.enabled
	name := c.name
	baseFields := c.fields
	c.mu.Unlock()

	if !enabled {
		return nil
	}

	// Convert fields to map
	fieldsMap := make(map[string]interface{})
	allFields := append(baseFields, fields...)
	for _, f := range allFields {
		fieldsMap[f.Key] = fieldToInterface(f)
	}

	// Determine source from logger name or caller
	source := name
	if source == "" && entry.LoggerName != "" {
		source = entry.LoggerName
	}
	if source == "" && entry.Caller.Defined {
		source = entry.Caller.TrimmedPath()
	}

	logEntry := LogEntry{
		Level:     levelToString(entry.Level),
		Message:   entry.Message,
		Source:    source,
		Timestamp: entry.Time.UnixMilli(),
	}

	if len(fieldsMap) > 0 {
		logEntry.Fields = fieldsMap
	}

	// Add to buffer (always, for polling)
	c.addToBuffer(logEntry)

	// Try to emit via push (may not work in Energy IPC)
	if emitter != nil {
		emitter(logEntry)
	}

	return nil
}

// addToBuffer adds a log entry to the circular buffer
func (c *IPCCore) addToBuffer(entry LogEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastID++
	c.buffer = append(c.buffer, entry)

	// Keep only last bufferMax entries
	if len(c.buffer) > c.bufferMax {
		c.buffer = c.buffer[len(c.buffer)-c.bufferMax:]
	}
}

// GetLogs returns logs from the buffer, optionally since a specific timestamp
func (c *IPCCore) GetLogs(sinceTimestamp int64, limit int) []LogEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 || limit > len(c.buffer) {
		limit = len(c.buffer)
	}

	result := make([]LogEntry, 0, limit)

	for i := len(c.buffer) - 1; i >= 0 && len(result) < limit; i-- {
		entry := c.buffer[i]
		if sinceTimestamp > 0 && entry.Timestamp <= sinceTimestamp {
			break
		}
		result = append(result, entry)
	}

	// Reverse to get chronological order
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// GetAllLogs returns all logs from the buffer
func (c *IPCCore) GetAllLogs() []LogEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]LogEntry, len(c.buffer))
	copy(result, c.buffer)
	return result
}

// ClearBuffer clears the log buffer
func (c *IPCCore) ClearBuffer() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.buffer = make([]LogEntry, 0, c.bufferMax)
}

// Sync flushes any buffered log entries
func (c *IPCCore) Sync() error {
	return nil
}

// SetEnabled enables or disables the IPC core
func (c *IPCCore) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = enabled
}

// SetLevel sets the log level for the IPC core
func (c *IPCCore) SetLevel(level zapcore.Level) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.level = level
}

// SetEmitter sets the log emitter function
func (c *IPCCore) SetEmitter(emitter LogEmitter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.emitter = emitter
}

// levelToString converts zapcore.Level to string
func levelToString(level zapcore.Level) string {
	switch level {
	case zapcore.DebugLevel:
		return "debug"
	case zapcore.InfoLevel:
		return "info"
	case zapcore.WarnLevel:
		return "warn"
	case zapcore.ErrorLevel:
		return "error"
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return "error"
	default:
		return "info"
	}
}

// fieldToInterface converts a zapcore.Field to an interface{}
func fieldToInterface(f zapcore.Field) interface{} {
	switch f.Type {
	case zapcore.BoolType:
		return f.Integer == 1
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return f.Integer
	case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
		return uint64(f.Integer)
	case zapcore.Float64Type:
		return f.Interface
	case zapcore.Float32Type:
		return f.Interface
	case zapcore.StringType:
		return f.String
	case zapcore.DurationType:
		return time.Duration(f.Integer).String()
	case zapcore.TimeType:
		if f.Interface != nil {
			return f.Interface.(time.Time).UnixMilli()
		}
		return time.Unix(0, f.Integer).UnixMilli()
	case zapcore.ErrorType:
		if f.Interface != nil {
			return f.Interface.(error).Error()
		}
		return nil
	default:
		if f.Interface != nil {
			return f.Interface
		}
		return f.String
	}
}

// IPCLoggerConfig contains configuration for IPC-enabled logger
type IPCLoggerConfig struct {
	Config
	IPCEmitter LogEmitter
}

// NewWithIPCConfig creates a new logger with IPC support
func NewWithIPCConfig(cfg IPCLoggerConfig) (*Logger, *IPCCore) {
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

	// IPC core for frontend
	var ipcCore *IPCCore
	if cfg.IPCEmitter != nil {
		ipcCore = NewIPCCore(level, cfg.IPCEmitter)
		cores = append(cores, ipcCore)
	}

	// Combine cores
	core := zapcore.NewTee(cores...)

	// Build logger
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &Logger{
		SugaredLogger: zapLogger.Sugar(),
		underlying:    zapLogger,
	}, ipcCore
}
