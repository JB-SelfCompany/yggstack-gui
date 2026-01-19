package ipc

import (
	gocontext "context"
	"encoding/json"
	"sync"
	"time"

	"github.com/energye/energy/v2/cef/ipc"
	ipccontext "github.com/energye/energy/v2/cef/ipc/context"
	"github.com/energye/energy/v2/cef/ipc/target"
	"github.com/JB-SelfCompany/yggstack-gui/internal/logger"
)

// Bridge handles IPC communication between Go backend and JS frontend
type Bridge struct {
	mu          sync.RWMutex
	logger      *logger.Logger
	handlers    map[string]Handler
	subscribers map[string][]Subscriber
	middleware  []Middleware
	timeout     time.Duration
}

// Handler is a function that handles an IPC event
type Handler func(request *Request) *Response

// Subscriber is a function that receives events
type Subscriber func(event string, data interface{})

// Middleware processes requests before handlers
type Middleware func(event string, req *Request, next func() *Response) *Response

// Request represents an IPC request from the frontend
type Request struct {
	RequestID string          `json:"requestId"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp,omitempty"`
}

// Response represents an IPC response to the frontend
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *Error      `json:"error,omitempty"`
	RequestID string      `json:"requestId"`
	Timestamp int64       `json:"timestamp,omitempty"`
}

// Error represents an error in the response
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Event represents a push event from backend to frontend
type Event struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// BridgeConfig contains bridge configuration
type BridgeConfig struct {
	Timeout time.Duration
	Logger  *logger.Logger
}

// DefaultBridgeConfig returns default configuration
func DefaultBridgeConfig() BridgeConfig {
	return BridgeConfig{
		Timeout: 30 * time.Second,
	}
}

// NewBridge creates a new IPC bridge
func NewBridge(log *logger.Logger) *Bridge {
	return NewBridgeWithConfig(BridgeConfig{
		Timeout: 30 * time.Second,
		Logger:  log,
	})
}

// NewBridgeWithConfig creates a new IPC bridge with configuration
func NewBridgeWithConfig(cfg BridgeConfig) *Bridge {
	return &Bridge{
		logger:      cfg.Logger,
		handlers:    make(map[string]Handler),
		subscribers: make(map[string][]Subscriber),
		middleware:  make([]Middleware, 0),
		timeout:     cfg.Timeout,
	}
}

// Use adds middleware to the bridge
func (b *Bridge) Use(mw Middleware) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.middleware = append(b.middleware, mw)
}

// Register adds a handler for an event
func (b *Bridge) Register(event string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[event] = handler

	// Register with Energy IPC
	ipc.On(event, func(ctx ipccontext.IContext) {
		b.handleEvent(event, ctx)
	})

	b.logger.Debug("Registered IPC handler", "event", event)
}

// Subscribe adds a subscriber for an event type
func (b *Bridge) Subscribe(event string, subscriber Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscribers[event]; !exists {
		b.subscribers[event] = make([]Subscriber, 0)
	}
	b.subscribers[event] = append(b.subscribers[event], subscriber)
}

// handleEvent processes an incoming IPC event
func (b *Bridge) handleEvent(event string, ctx ipccontext.IContext) {
	startTime := time.Now()

	b.mu.RLock()
	handler, exists := b.handlers[event]
	middleware := make([]Middleware, len(b.middleware))
	copy(middleware, b.middleware)
	b.mu.RUnlock()

	if !exists {
		b.logger.Warn("No handler for event", "event", event)
		b.sendError(ctx, "", "HANDLER_NOT_FOUND", "No handler registered for event: "+event)
		return
	}

	// Parse request
	var req Request
	args := ctx.ArgumentList()
	if args.Size() > 0 {
		data := args.GetStringByIndex(0)
		if err := json.Unmarshal([]byte(data), &req); err != nil {
			b.logger.Error("Failed to parse request", "event", event, "error", err)
			b.sendError(ctx, "", "PARSE_ERROR", "Failed to parse request: "+err.Error())
			return
		}
	}

	req.Timestamp = time.Now().UnixMilli()
	b.logger.Debug("Handling IPC event", "event", event, "requestId", req.RequestID)

	// Create handler chain with middleware
	var response *Response

	// Build the handler chain
	finalHandler := func() *Response {
		return handler(&req)
	}

	// Apply middleware in reverse order
	chain := finalHandler
	for i := len(middleware) - 1; i >= 0; i-- {
		mw := middleware[i]
		next := chain
		chain = func() *Response {
			return mw(event, &req, next)
		}
	}

	// Execute with timeout
	done := make(chan struct{})
	go func() {
		response = chain()
		close(done)
	}()

	select {
	case <-done:
		// Handler completed
	case <-time.After(b.timeout):
		b.logger.Warn("Handler timeout", "event", event, "timeout", b.timeout)
		response = &Response{
			Success: false,
			Error: &Error{
				Code:    "TIMEOUT",
				Message: "Request timed out",
			},
		}
	}

	response.RequestID = req.RequestID
	response.Timestamp = time.Now().UnixMilli()

	// Log performance
	duration := time.Since(startTime)
	if duration > 100*time.Millisecond {
		b.logger.Warn("Slow IPC handler", "event", event, "duration", duration)
	}

	// Send response
	b.sendResponse(ctx, response)
}

// sendResponse sends a response back to the frontend
func (b *Bridge) sendResponse(ctx ipccontext.IContext, response *Response) {
	data, err := json.Marshal(response)
	if err != nil {
		b.logger.Error("Failed to marshal response", "error", err)
		return
	}

	ctx.Result(string(data))
}

// sendError sends an error response
func (b *Bridge) sendError(ctx ipccontext.IContext, requestID, code, message string) {
	response := &Response{
		Success:   false,
		RequestID: requestID,
		Timestamp: time.Now().UnixMilli(),
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
	b.sendResponse(ctx, response)
}

// Emit sends an event to the frontend
func (b *Bridge) Emit(event string, data interface{}) {
	evt := Event{
		Type:      event,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		// Don't log for log:entry to avoid infinite loop
		if event != EventLogEntry {
			b.logger.Error("Failed to marshal emit data", "event", event, "error", err)
		}
		return
	}

	// Try to emit via Energy IPC
	ipc.Emit(event, string(payload))

	// Don't log for log:entry to avoid infinite loop
	if event != EventLogEntry {
		b.logger.Debug("Emitted event", "event", event, "payloadLen", len(payload))
	}

	// Notify subscribers
	b.notifySubscribers(event, data)
}

// EmitToWindow sends an event to a specific browser window
func (b *Bridge) EmitToWindow(browserId int32, event string, data interface{}) {
	evt := Event{
		Type:      event,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		b.logger.Error("Failed to marshal emit data", "event", event, "error", err)
		return
	}

	t := target.NewTarget(nil, browserId, "")
	ipc.EmitTarget(event, t, string(payload))
	b.logger.Debug("Emitted event to window", "event", event, "browserId", browserId)
}

// notifySubscribers notifies all subscribers of an event
func (b *Bridge) notifySubscribers(event string, data interface{}) {
	b.mu.RLock()
	subs, exists := b.subscribers[event]
	b.mu.RUnlock()

	if !exists || len(subs) == 0 {
		return
	}

	for _, sub := range subs {
		go sub(event, data)
	}
}

// RegisterHandlers registers core IPC event handlers (backward compatibility)
func (b *Bridge) RegisterHandlers() {
	b.logger.Info("Registering core IPC handlers")

	// Register core handlers
	b.Register("app:version", b.handleAppVersion)
	b.Register("app:ready", b.handleAppReady)
	b.Register("app:ping", b.handlePing)

	b.logger.Info("Core IPC handlers registered")
}

// Core handler implementations

func (b *Bridge) handleAppVersion(req *Request) *Response {
	return &Response{
		Success: true,
		Data: map[string]string{
			"version": "0.1.0-dev",
		},
	}
}

func (b *Bridge) handleAppReady(req *Request) *Response {
	b.logger.Info("Frontend reported ready")
	return &Response{
		Success: true,
		Data:    map[string]bool{"acknowledged": true},
	}
}

func (b *Bridge) handlePing(req *Request) *Response {
	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"pong":      true,
			"timestamp": time.Now().UnixMilli(),
		},
	}
}

// EventEmitter provides a convenient way to emit state updates
type EventEmitter struct {
	bridge *Bridge
	prefix string
}

// NewEventEmitter creates an event emitter with a prefix
func NewEventEmitter(bridge *Bridge, prefix string) *EventEmitter {
	return &EventEmitter{
		bridge: bridge,
		prefix: prefix,
	}
}

// Emit sends an event with the emitter's prefix
func (e *EventEmitter) Emit(event string, data interface{}) {
	fullEvent := event
	if e.prefix != "" {
		fullEvent = e.prefix + ":" + event
	}
	e.bridge.Emit(fullEvent, data)
}

// StateSync manages state synchronization between backend and frontend
type StateSync struct {
	mu       sync.RWMutex
	bridge   *Bridge
	states   map[string]interface{}
	watchers map[string][]func(interface{})
	logger   *logger.Logger
}

// NewStateSync creates a new state synchronizer
func NewStateSync(bridge *Bridge, log *logger.Logger) *StateSync {
	return &StateSync{
		bridge:   bridge,
		states:   make(map[string]interface{}),
		watchers: make(map[string][]func(interface{})),
		logger:   log,
	}
}

// Set updates a state value and notifies the frontend
func (s *StateSync) Set(key string, value interface{}) {
	s.mu.Lock()
	s.states[key] = value
	watchers := s.watchers[key]
	s.mu.Unlock()

	// Emit state change to frontend
	s.bridge.Emit("state:changed", map[string]interface{}{
		"key":   key,
		"value": value,
	})

	// Notify watchers
	for _, watcher := range watchers {
		go watcher(value)
	}

	s.logger.Debug("State updated", "key", key)
}

// Get retrieves a state value
func (s *StateSync) Get(key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.states[key]
}

// Watch adds a watcher for state changes
func (s *StateSync) Watch(key string, callback func(interface{})) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.watchers[key]; !exists {
		s.watchers[key] = make([]func(interface{}), 0)
	}
	s.watchers[key] = append(s.watchers[key], callback)
}

// GetAll returns all current states
func (s *StateSync) GetAll() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range s.states {
		result[k] = v
	}
	return result
}

// SyncAll sends all current states to the frontend
func (s *StateSync) SyncAll() {
	s.bridge.Emit("state:sync", s.GetAll())
}

// RequestContext provides context for IPC requests
type RequestContext struct {
	gocontext.Context
	Request *Request
	Bridge  *Bridge
}

// NewRequestContext creates a new request context
func NewRequestContext(ctx gocontext.Context, req *Request, bridge *Bridge) *RequestContext {
	return &RequestContext{
		Context: ctx,
		Request: req,
		Bridge:  bridge,
	}
}

// ParsePayload parses the request payload into the given target
func (c *RequestContext) ParsePayload(target interface{}) error {
	return json.Unmarshal(c.Request.Payload, target)
}

// Success creates a success response
func (c *RequestContext) Success(data interface{}) *Response {
	return &Response{
		Success: true,
		Data:    data,
	}
}

// Fail creates an error response
func (c *RequestContext) Fail(code, message string) *Response {
	return &Response{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
}
