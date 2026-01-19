package ipc

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestRequestJSON(t *testing.T) {
	req := Request{
		RequestID: "req-123",
		Payload:   json.RawMessage(`{"key": "value"}`),
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal Request: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Request: %v", err)
	}

	if decoded.RequestID != req.RequestID {
		t.Errorf("RequestID = %q, want %q", decoded.RequestID, req.RequestID)
	}
	if string(decoded.Payload) != string(req.Payload) {
		t.Errorf("Payload = %q, want %q", string(decoded.Payload), string(req.Payload))
	}
}

func TestResponseJSON(t *testing.T) {
	tests := []struct {
		name     string
		response Response
	}{
		{
			name: "success response",
			response: Response{
				Success:   true,
				Data:      map[string]string{"status": "ok"},
				RequestID: "req-123",
				Timestamp: time.Now().UnixMilli(),
			},
		},
		{
			name: "error response",
			response: Response{
				Success:   false,
				Error:     &Error{Code: "ERR_001", Message: "Something went wrong"},
				RequestID: "req-456",
				Timestamp: time.Now().UnixMilli(),
			},
		},
		{
			name: "response with nil data",
			response: Response{
				Success:   true,
				RequestID: "req-789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.response)
			if err != nil {
				t.Fatalf("Failed to marshal Response: %v", err)
			}

			var decoded Response
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal Response: %v", err)
			}

			if decoded.Success != tt.response.Success {
				t.Errorf("Success = %v, want %v", decoded.Success, tt.response.Success)
			}
			if decoded.RequestID != tt.response.RequestID {
				t.Errorf("RequestID = %q, want %q", decoded.RequestID, tt.response.RequestID)
			}
		})
	}
}

func TestErrorJSON(t *testing.T) {
	err := Error{
		Code:    "VALIDATION_ERROR",
		Message: "Invalid input",
		Details: "Field 'uri' is required",
	}

	data, e := json.Marshal(err)
	if e != nil {
		t.Fatalf("Failed to marshal Error: %v", e)
	}

	var decoded Error
	if e := json.Unmarshal(data, &decoded); e != nil {
		t.Fatalf("Failed to unmarshal Error: %v", e)
	}

	if decoded.Code != err.Code {
		t.Errorf("Code = %q, want %q", decoded.Code, err.Code)
	}
	if decoded.Message != err.Message {
		t.Errorf("Message = %q, want %q", decoded.Message, err.Message)
	}
	if decoded.Details != err.Details {
		t.Errorf("Details = %q, want %q", decoded.Details, err.Details)
	}
}

func TestEventJSON(t *testing.T) {
	event := Event{
		Type:      "node:stateChanged",
		Data:      map[string]string{"state": "running"},
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal Event: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Event: %v", err)
	}

	if decoded.Type != event.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, event.Type)
	}
}

func TestDefaultBridgeConfig(t *testing.T) {
	cfg := DefaultBridgeConfig()

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 30*time.Second)
	}
}

func TestRequestContextParsePayload(t *testing.T) {
	type TestPayload struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	req := &Request{
		RequestID: "req-123",
		Payload:   json.RawMessage(`{"name": "test", "value": 42}`),
	}

	ctx := NewRequestContext(context.Background(), req, nil)

	var payload TestPayload
	if err := ctx.ParsePayload(&payload); err != nil {
		t.Fatalf("ParsePayload error: %v", err)
	}

	if payload.Name != "test" {
		t.Errorf("Name = %q, want %q", payload.Name, "test")
	}
	if payload.Value != 42 {
		t.Errorf("Value = %d, want %d", payload.Value, 42)
	}
}

func TestRequestContextParsePayloadError(t *testing.T) {
	req := &Request{
		RequestID: "req-123",
		Payload:   json.RawMessage(`invalid json`),
	}

	ctx := NewRequestContext(context.Background(), req, nil)

	var payload map[string]string
	err := ctx.ParsePayload(&payload)
	if err == nil {
		t.Error("ParsePayload should error for invalid JSON")
	}
}

func TestRequestContextSuccess(t *testing.T) {
	ctx := NewRequestContext(context.Background(), &Request{}, nil)

	data := map[string]string{"status": "ok"}
	resp := ctx.Success(data)

	if !resp.Success {
		t.Error("Success response should have Success=true")
	}
	if resp.Error != nil {
		t.Error("Success response should not have Error")
	}
	if resp.Data == nil {
		t.Error("Success response should have Data")
	}
}

func TestRequestContextFail(t *testing.T) {
	ctx := NewRequestContext(context.Background(), &Request{}, nil)

	resp := ctx.Fail("TEST_ERROR", "Test error message")

	if resp.Success {
		t.Error("Fail response should have Success=false")
	}
	if resp.Error == nil {
		t.Fatal("Fail response should have Error")
	}
	if resp.Error.Code != "TEST_ERROR" {
		t.Errorf("Error.Code = %q, want %q", resp.Error.Code, "TEST_ERROR")
	}
	if resp.Error.Message != "Test error message" {
		t.Errorf("Error.Message = %q, want %q", resp.Error.Message, "Test error message")
	}
}

func TestStateSyncConcurrent(t *testing.T) {
	// Create mock StateSync without bridge dependency
	type mockStateSync struct {
		mu     sync.RWMutex
		states map[string]interface{}
	}

	ss := &mockStateSync{
		states: make(map[string]interface{}),
	}

	set := func(key string, value interface{}) {
		ss.mu.Lock()
		defer ss.mu.Unlock()
		ss.states[key] = value
	}

	get := func(key string) interface{} {
		ss.mu.RLock()
		defer ss.mu.RUnlock()
		return ss.states[key]
	}

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			set("key", i)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = get("key")
		}()
	}

	wg.Wait()

	// Should have some value
	val := get("key")
	if val == nil {
		t.Error("State should have a value after concurrent access")
	}
}

func TestEventEmitterPrefix(t *testing.T) {
	// Test EventEmitter with prefix logic (without actual bridge)
	type mockEmitter struct {
		prefix    string
		lastEvent string
	}

	emit := func(e *mockEmitter, event string) string {
		fullEvent := event
		if e.prefix != "" {
			fullEvent = e.prefix + ":" + event
		}
		return fullEvent
	}

	tests := []struct {
		prefix   string
		event    string
		expected string
	}{
		{"", "test", "test"},
		{"node", "status", "node:status"},
		{"peers", "update", "peers:update"},
	}

	for _, tt := range tests {
		e := &mockEmitter{prefix: tt.prefix}
		result := emit(e, tt.event)
		if result != tt.expected {
			t.Errorf("emit(%q, %q) = %q, want %q", tt.prefix, tt.event, result, tt.expected)
		}
	}
}

// Benchmark tests
func BenchmarkRequestJSON(b *testing.B) {
	req := Request{
		RequestID: "req-123",
		Payload:   json.RawMessage(`{"uri": "tls://example.com:443"}`),
		Timestamp: time.Now().UnixMilli(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(req)
		var decoded Request
		json.Unmarshal(data, &decoded)
	}
}

func BenchmarkResponseJSON(b *testing.B) {
	resp := Response{
		Success:   true,
		Data:      map[string]interface{}{"status": "ok", "count": 42},
		RequestID: "req-123",
		Timestamp: time.Now().UnixMilli(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(resp)
		var decoded Response
		json.Unmarshal(data, &decoded)
	}
}

func BenchmarkParsePayload(b *testing.B) {
	type Payload struct {
		URI     string `json:"uri"`
		Enabled bool   `json:"enabled"`
	}

	req := &Request{
		Payload: json.RawMessage(`{"uri": "tls://example.com:443", "enabled": true}`),
	}
	ctx := NewRequestContext(context.Background(), req, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var p Payload
		ctx.ParsePayload(&p)
	}
}
