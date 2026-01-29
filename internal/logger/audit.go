package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// AuditEventType represents the type of security event
type AuditEventType string

const (
	// Authentication events
	AuditEventNodeStart     AuditEventType = "NODE_START"
	AuditEventNodeStop      AuditEventType = "NODE_STOP"
	AuditEventConfigLoad    AuditEventType = "CONFIG_LOAD"
	AuditEventConfigSave    AuditEventType = "CONFIG_SAVE"
	AuditEventConfigChange  AuditEventType = "CONFIG_CHANGE"

	// Peer events
	AuditEventPeerAdd       AuditEventType = "PEER_ADD"
	AuditEventPeerRemove    AuditEventType = "PEER_REMOVE"
	AuditEventPeerConnect   AuditEventType = "PEER_CONNECT"
	AuditEventPeerDisconnect AuditEventType = "PEER_DISCONNECT"

	// Security events
	AuditEventKeyAccess     AuditEventType = "KEY_ACCESS"
	AuditEventKeyStore      AuditEventType = "KEY_STORE"
	AuditEventKeyDelete     AuditEventType = "KEY_DELETE"
	AuditEventValidationFail AuditEventType = "VALIDATION_FAIL"
	AuditEventSecurityError AuditEventType = "SECURITY_ERROR"

	// IPC events
	AuditEventIPCRequest    AuditEventType = "IPC_REQUEST"
	AuditEventIPCError      AuditEventType = "IPC_ERROR"
	AuditEventIPCRateLimit  AuditEventType = "IPC_RATE_LIMIT"

	// Proxy/Mapping events
	AuditEventProxyStart    AuditEventType = "PROXY_START"
	AuditEventProxyStop     AuditEventType = "PROXY_STOP"
	AuditEventMappingAdd    AuditEventType = "MAPPING_ADD"
	AuditEventMappingRemove AuditEventType = "MAPPING_REMOVE"

	// Application events
	AuditEventAppStart      AuditEventType = "APP_START"
	AuditEventAppStop       AuditEventType = "APP_STOP"
	AuditEventSettingsChange AuditEventType = "SETTINGS_CHANGE"
)

// AuditSeverity represents the severity level of an audit event
type AuditSeverity string

const (
	SeverityInfo     AuditSeverity = "INFO"
	SeverityWarning  AuditSeverity = "WARNING"
	SeverityCritical AuditSeverity = "CRITICAL"
)

// AuditEvent represents a security audit event
type AuditEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   AuditEventType         `json:"event_type"`
	Severity    AuditSeverity          `json:"severity"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Source      string                 `json:"source,omitempty"`
	Result      string                 `json:"result"` // "success" or "failure"
	Error       string                 `json:"error,omitempty"`
}

// AuditLogger handles security audit logging
type AuditLogger struct {
	mu         sync.Mutex
	logger     *zap.Logger
	filePath   string
	file       *os.File
	maxSize    int64 // Maximum file size in bytes
	rotated    int   // Number of rotated files
	maxRotated int   // Maximum number of rotated files
}

// AuditConfig holds audit logger configuration
type AuditConfig struct {
	FilePath   string // Path to audit log file
	MaxSize    int64  // Maximum file size before rotation (default 10MB)
	MaxRotated int    // Maximum number of rotated files (default 5)
	Console    bool   // Also log to console
}

// DefaultAuditConfig returns default audit configuration
func DefaultAuditConfig() AuditConfig {
	return AuditConfig{
		FilePath:   GetAuditLogPath(),
		MaxSize:    10 * 1024 * 1024, // 10MB
		MaxRotated: 5,
		Console:    false,
	}
}

// GetAuditLogPath returns the default audit log file path
// PORTABLE MODE: audit logs are stored in "data/logs" subdirectory next to the executable
func GetAuditLogPath() string {
	// Get executable directory for portable mode
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to current working directory
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, "data", "logs", "audit.log")
	}
	// Resolve symlinks to get the real path
	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return filepath.Join(filepath.Dir(exePath), "data", "logs", "audit.log")
	}
	return filepath.Join(filepath.Dir(realPath), "data", "logs", "audit.log")
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(cfg AuditConfig) (*AuditLogger, error) {
	al := &AuditLogger{
		filePath:   cfg.FilePath,
		maxSize:    cfg.MaxSize,
		maxRotated: cfg.MaxRotated,
	}

	// Ensure directory exists
	dir := filepath.Dir(cfg.FilePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	// Open log file
	file, err := os.OpenFile(cfg.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	al.file = file

	// Create zap logger
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	var cores []zapcore.Core

	// File output (JSON format for machine parsing)
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
	fileCore := zapcore.NewCore(
		fileEncoder,
		zapcore.AddSync(file),
		zapcore.InfoLevel,
	)
	cores = append(cores, fileCore)

	// Optional console output
	if cfg.Console {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		consoleCore := zapcore.NewCore(
			consoleEncoder,
			zapcore.AddSync(os.Stdout),
			zapcore.InfoLevel,
		)
		cores = append(cores, consoleCore)
	}

	al.logger = zap.New(zapcore.NewTee(cores...))

	return al, nil
}

// Log records an audit event
func (al *AuditLogger) Log(event AuditEvent) {
	al.mu.Lock()
	defer al.mu.Unlock()

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Check if rotation is needed
	al.checkRotation()

	// Convert to JSON for structured logging
	fields := []zap.Field{
		zap.String("event_type", string(event.EventType)),
		zap.String("severity", string(event.Severity)),
		zap.String("result", event.Result),
	}

	if event.Source != "" {
		fields = append(fields, zap.String("source", event.Source))
	}

	if event.Error != "" {
		fields = append(fields, zap.String("error", event.Error))
	}

	if event.Details != nil {
		// Redact sensitive fields
		redactedDetails := al.redactSensitive(event.Details)
		detailsJSON, _ := json.Marshal(redactedDetails)
		fields = append(fields, zap.String("details", string(detailsJSON)))
	}

	// Log at appropriate level
	switch event.Severity {
	case SeverityCritical:
		al.logger.Error(event.Description, fields...)
	case SeverityWarning:
		al.logger.Warn(event.Description, fields...)
	default:
		al.logger.Info(event.Description, fields...)
	}
}

// LogSuccess logs a successful security event
func (al *AuditLogger) LogSuccess(eventType AuditEventType, description string, details map[string]interface{}) {
	al.Log(AuditEvent{
		EventType:   eventType,
		Severity:    SeverityInfo,
		Description: description,
		Details:     details,
		Result:      "success",
	})
}

// LogFailure logs a failed security event
func (al *AuditLogger) LogFailure(eventType AuditEventType, description string, err error, details map[string]interface{}) {
	event := AuditEvent{
		EventType:   eventType,
		Severity:    SeverityWarning,
		Description: description,
		Details:     details,
		Result:      "failure",
	}

	if err != nil {
		event.Error = err.Error()
	}

	al.Log(event)
}

// LogCritical logs a critical security event
func (al *AuditLogger) LogCritical(eventType AuditEventType, description string, err error, details map[string]interface{}) {
	event := AuditEvent{
		EventType:   eventType,
		Severity:    SeverityCritical,
		Description: description,
		Details:     details,
		Result:      "failure",
	}

	if err != nil {
		event.Error = err.Error()
	}

	al.Log(event)
}

// redactSensitive removes sensitive information from details
func (al *AuditLogger) redactSensitive(details map[string]interface{}) map[string]interface{} {
	sensitiveKeys := map[string]bool{
		"password":    true,
		"secret":      true,
		"key":         true,
		"privateKey":  true,
		"private_key": true,
		"token":       true,
		"credential":  true,
	}

	redacted := make(map[string]interface{})
	for k, v := range details {
		if sensitiveKeys[k] {
			redacted[k] = "[REDACTED]"
		} else {
			// Check if value is a string that looks like a key
			if str, ok := v.(string); ok && len(str) == 64 {
				// Might be a public key, redact partially
				if len(str) > 16 {
					redacted[k] = str[:8] + "..." + str[len(str)-8:]
				} else {
					redacted[k] = "[REDACTED]"
				}
			} else {
				redacted[k] = v
			}
		}
	}

	return redacted
}

// checkRotation checks if log rotation is needed
func (al *AuditLogger) checkRotation() {
	if al.file == nil {
		return
	}

	info, err := al.file.Stat()
	if err != nil {
		return
	}

	if info.Size() < al.maxSize {
		return
	}

	// Rotate logs
	al.rotate()
}

// rotate rotates the audit log file
func (al *AuditLogger) rotate() {
	if al.file != nil {
		al.file.Close()
	}

	// Shift existing rotated files
	for i := al.maxRotated - 1; i > 0; i-- {
		oldPath := al.filePath + "." + string(rune('0'+i))
		newPath := al.filePath + "." + string(rune('0'+i+1))
		os.Rename(oldPath, newPath)
	}

	// Rename current file to .1
	os.Rename(al.filePath, al.filePath+".1")

	// Create new file
	file, err := os.OpenFile(al.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}

	al.file = file
	al.rotated++

	// Rebuild zap logger with new file
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		MessageKey:     "message",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
	fileCore := zapcore.NewCore(
		fileEncoder,
		zapcore.AddSync(file),
		zapcore.InfoLevel,
	)

	al.logger = zap.New(fileCore)
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if al.logger != nil {
		al.logger.Sync()
	}

	if al.file != nil {
		return al.file.Close()
	}

	return nil
}

// Flush ensures all pending logs are written
func (al *AuditLogger) Flush() {
	al.mu.Lock()
	defer al.mu.Unlock()

	if al.logger != nil {
		al.logger.Sync()
	}

	if al.file != nil {
		al.file.Sync()
	}
}
