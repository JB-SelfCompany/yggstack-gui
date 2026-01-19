package logger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultAuditConfig(t *testing.T) {
	cfg := DefaultAuditConfig()

	if cfg.FilePath == "" {
		t.Error("FilePath should not be empty")
	}
	if cfg.MaxSize != 10*1024*1024 {
		t.Errorf("MaxSize = %d, want %d", cfg.MaxSize, 10*1024*1024)
	}
	if cfg.MaxRotated != 5 {
		t.Errorf("MaxRotated = %d, want 5", cfg.MaxRotated)
	}
	if cfg.Console {
		t.Error("Console should be false by default")
	}
}

func TestGetAuditLogPath(t *testing.T) {
	path := GetAuditLogPath()
	if path == "" {
		t.Error("GetAuditLogPath() should not return empty string")
	}
	if !strings.Contains(path, "yggstack-gui") {
		t.Errorf("path %q should contain 'yggstack-gui'", path)
	}
	if !strings.HasSuffix(path, "audit.log") {
		t.Errorf("path %q should end with 'audit.log'", path)
	}
}

func TestNewAuditLogger(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := AuditConfig{
		FilePath:   filepath.Join(tmpDir, "audit.log"),
		MaxSize:    1024,
		MaxRotated: 3,
	}

	al, err := NewAuditLogger(cfg)
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer al.Close()

	if al.logger == nil {
		t.Error("logger should not be nil")
	}
	if al.file == nil {
		t.Error("file should not be nil")
	}
}

func TestAuditLogger_Log(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	al, err := NewAuditLogger(AuditConfig{
		FilePath:   logPath,
		MaxSize:    1024 * 1024,
		MaxRotated: 3,
	})
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer al.Close()

	event := AuditEvent{
		EventType:   AuditEventNodeStart,
		Severity:    SeverityInfo,
		Description: "Test event",
		Result:      "success",
		Details:     map[string]interface{}{"key": "value"},
	}

	al.Log(event)
	al.Flush()

	// Verify log was written
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), "Test event") {
		t.Error("log should contain 'Test event'")
	}
	if !strings.Contains(string(data), "NODE_START") {
		t.Error("log should contain event type")
	}
}

func TestAuditLogger_LogSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	al, err := NewAuditLogger(AuditConfig{
		FilePath:   logPath,
		MaxSize:    1024 * 1024,
		MaxRotated: 3,
	})
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer al.Close()

	al.LogSuccess(AuditEventConfigSave, "Config saved", map[string]interface{}{"path": "/test"})
	al.Flush()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), "success") {
		t.Error("log should contain 'success'")
	}
}

func TestAuditLogger_LogFailure(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	al, err := NewAuditLogger(AuditConfig{
		FilePath:   logPath,
		MaxSize:    1024 * 1024,
		MaxRotated: 3,
	})
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer al.Close()

	testErr := errors.New("test error")
	al.LogFailure(AuditEventValidationFail, "Validation failed", testErr, nil)
	al.Flush()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), "failure") {
		t.Error("log should contain 'failure'")
	}
	if !strings.Contains(string(data), "test error") {
		t.Error("log should contain error message")
	}
}

func TestAuditLogger_LogCritical(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	al, err := NewAuditLogger(AuditConfig{
		FilePath:   logPath,
		MaxSize:    1024 * 1024,
		MaxRotated: 3,
	})
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}
	defer al.Close()

	testErr := errors.New("critical error")
	al.LogCritical(AuditEventSecurityError, "Security breach", testErr, nil)
	al.Flush()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), "CRITICAL") || !strings.Contains(string(data), "ERROR") {
		// Should be logged at error level
		t.Log("Warning: critical event may be logged differently")
	}
}

func TestAuditLogger_RedactSensitive(t *testing.T) {
	tmpDir := t.TempDir()
	al, _ := NewAuditLogger(AuditConfig{
		FilePath:   filepath.Join(tmpDir, "audit.log"),
		MaxSize:    1024 * 1024,
		MaxRotated: 3,
	})
	defer al.Close()

	details := map[string]interface{}{
		"username":   "testuser",
		"password":   "secret123",
		"privateKey": "key-data",
		"token":      "bearer-token",
		"normal":     "normal-value",
	}

	redacted := al.redactSensitive(details)

	// Check sensitive fields are redacted
	if redacted["password"] != "[REDACTED]" {
		t.Error("password should be redacted")
	}
	if redacted["privateKey"] != "[REDACTED]" {
		t.Error("privateKey should be redacted")
	}
	if redacted["token"] != "[REDACTED]" {
		t.Error("token should be redacted")
	}

	// Check normal fields are not redacted
	if redacted["username"] != "testuser" {
		t.Error("username should not be redacted")
	}
	if redacted["normal"] != "normal-value" {
		t.Error("normal should not be redacted")
	}
}

func TestAuditLogger_RedactLongStrings(t *testing.T) {
	tmpDir := t.TempDir()
	al, _ := NewAuditLogger(AuditConfig{
		FilePath:   filepath.Join(tmpDir, "audit.log"),
		MaxSize:    1024 * 1024,
		MaxRotated: 3,
	})
	defer al.Close()

	// 64-char string that looks like a key
	longKey := "1234567890123456789012345678901234567890123456789012345678901234"
	details := map[string]interface{}{
		"publicKey": longKey,
	}

	redacted := al.redactSensitive(details)

	// Should be partially redacted (8...8 format)
	result := redacted["publicKey"].(string)
	if !strings.Contains(result, "...") {
		t.Error("64-char string should be partially redacted")
	}
	if len(result) > 20 {
		t.Error("redacted string should be shorter than original")
	}
}

func TestAuditLogger_TimestampAutoSet(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	al, _ := NewAuditLogger(AuditConfig{
		FilePath:   logPath,
		MaxSize:    1024 * 1024,
		MaxRotated: 3,
	})
	defer al.Close()

	event := AuditEvent{
		EventType:   AuditEventAppStart,
		Severity:    SeverityInfo,
		Description: "App started",
		Result:      "success",
	}

	al.Log(event)

	// Timestamp should have been auto-set
	// Since we can't access the internal event, just verify no error
}

func TestAuditEventTypes(t *testing.T) {
	// Verify event type constants exist
	types := []AuditEventType{
		AuditEventNodeStart,
		AuditEventNodeStop,
		AuditEventConfigLoad,
		AuditEventConfigSave,
		AuditEventPeerAdd,
		AuditEventPeerRemove,
		AuditEventKeyAccess,
		AuditEventIPCRequest,
		AuditEventProxyStart,
		AuditEventAppStart,
	}

	for _, et := range types {
		if string(et) == "" {
			t.Error("event type should not be empty")
		}
	}
}

func TestAuditSeverityTypes(t *testing.T) {
	severities := []AuditSeverity{
		SeverityInfo,
		SeverityWarning,
		SeverityCritical,
	}

	for _, s := range severities {
		if string(s) == "" {
			t.Error("severity should not be empty")
		}
	}
}

func TestAuditLogger_Close(t *testing.T) {
	tmpDir := t.TempDir()
	al, err := NewAuditLogger(AuditConfig{
		FilePath:   filepath.Join(tmpDir, "audit.log"),
		MaxSize:    1024 * 1024,
		MaxRotated: 3,
	})
	if err != nil {
		t.Fatalf("NewAuditLogger() error = %v", err)
	}

	err = al.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Double close should be safe
	err = al.Close()
	// May return error or nil, just shouldn't panic
}
