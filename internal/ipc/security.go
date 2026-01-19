package ipc

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/JB-SelfCompany/yggstack-gui/internal/logger"
	"github.com/JB-SelfCompany/yggstack-gui/internal/security"
)

// SecurityConfig holds IPC security configuration
type SecurityConfig struct {
	EnableOriginValidation bool          // Validate request origins
	EnableRateLimiting     bool          // Enable request rate limiting
	MaxRequestsPerSecond   int           // Maximum requests per second per event
	MaxRequestSize         int           // Maximum request payload size in bytes
	EnableCSRFProtection   bool          // Enable CSRF token validation
	TokenExpiry            time.Duration // CSRF token expiry duration
}

// DefaultSecurityConfig returns secure defaults
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		EnableOriginValidation: true,
		EnableRateLimiting:     true,
		MaxRequestsPerSecond:   100,
		MaxRequestSize:         1024 * 1024, // 1MB
		EnableCSRFProtection:   true,
		TokenExpiry:            24 * time.Hour,
	}
}

// SecurityMiddleware provides security features for IPC
type SecurityMiddleware struct {
	mu             sync.RWMutex
	config         SecurityConfig
	validator      *security.Validator
	rateLimiter    *RateLimiter
	csrfTokens     map[string]*CSRFToken
	auditLogger    *logger.AuditLogger
	logger         *logger.Logger
}

// CSRFToken represents a CSRF protection token
type CSRFToken struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*tokenBucket
	maxTokens int
	refillRate time.Duration
}

type tokenBucket struct {
	tokens      int
	lastRefill  time.Time
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(cfg SecurityConfig, log *logger.Logger, auditLog *logger.AuditLogger) *SecurityMiddleware {
	sm := &SecurityMiddleware{
		config:      cfg,
		validator:   security.NewValidator(),
		csrfTokens:  make(map[string]*CSRFToken),
		auditLogger: auditLog,
		logger:      log,
	}

	if cfg.EnableRateLimiting {
		sm.rateLimiter = &RateLimiter{
			buckets:    make(map[string]*tokenBucket),
			maxTokens:  cfg.MaxRequestsPerSecond,
			refillRate: time.Second,
		}
	}

	// Start token cleanup goroutine
	go sm.cleanupExpiredTokens()

	return sm
}

// Middleware returns the middleware function for the bridge
func (sm *SecurityMiddleware) Middleware() Middleware {
	return func(event string, req *Request, next func() *Response) *Response {
		// Check request size
		if sm.config.MaxRequestSize > 0 && len(req.Payload) > sm.config.MaxRequestSize {
			sm.logSecurityEvent(logger.AuditEventIPCError, "Request too large", event, nil)
			return &Response{
				Success: false,
				Error: &Error{
					Code:    "REQUEST_TOO_LARGE",
					Message: "Request payload exceeds maximum size",
				},
			}
		}

		// Rate limiting
		if sm.config.EnableRateLimiting {
			if !sm.checkRateLimit(event) {
				sm.logSecurityEvent(logger.AuditEventIPCRateLimit, "Rate limit exceeded", event, nil)
				return &Response{
					Success: false,
					Error: &Error{
						Code:    "RATE_LIMITED",
						Message: "Too many requests, please try again later",
					},
				}
			}
		}

		// Validate JSON payload
		if len(req.Payload) > 0 {
			if err := sm.validator.ValidateJSON(string(req.Payload), sm.config.MaxRequestSize); err != nil {
				sm.logSecurityEvent(logger.AuditEventValidationFail, "Invalid JSON payload", event, err)
				return &Response{
					Success: false,
					Error: &Error{
						Code:    "INVALID_PAYLOAD",
						Message: "Invalid request payload",
					},
				}
			}
		}

		// Log IPC request
		sm.logSecurityEvent(logger.AuditEventIPCRequest, "IPC request", event, nil)

		// Execute handler
		resp := next()

		return resp
	}
}

// GenerateCSRFToken generates a new CSRF token
func (sm *SecurityMiddleware) GenerateCSRFToken() (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Generate random token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	token := hex.EncodeToString(bytes)

	sm.csrfTokens[token] = &CSRFToken{
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sm.config.TokenExpiry),
	}

	sm.logger.Debug("Generated CSRF token")
	return token, nil
}

// ValidateCSRFToken validates a CSRF token
func (sm *SecurityMiddleware) ValidateCSRFToken(token string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	csrfToken, exists := sm.csrfTokens[token]
	if !exists {
		return false
	}

	if time.Now().After(csrfToken.ExpiresAt) {
		return false
	}

	return true
}

// InvalidateCSRFToken invalidates a CSRF token
func (sm *SecurityMiddleware) InvalidateCSRFToken(token string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.csrfTokens, token)
}

// checkRateLimit checks if the request is within rate limits
func (sm *SecurityMiddleware) checkRateLimit(event string) bool {
	if sm.rateLimiter == nil {
		return true
	}

	sm.rateLimiter.mu.Lock()
	defer sm.rateLimiter.mu.Unlock()

	now := time.Now()
	bucket, exists := sm.rateLimiter.buckets[event]

	if !exists {
		bucket = &tokenBucket{
			tokens:     sm.rateLimiter.maxTokens,
			lastRefill: now,
		}
		sm.rateLimiter.buckets[event] = bucket
	}

	// Refill tokens
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed / sm.rateLimiter.refillRate) * sm.rateLimiter.maxTokens
	if tokensToAdd > 0 {
		bucket.tokens = min(bucket.tokens+tokensToAdd, sm.rateLimiter.maxTokens)
		bucket.lastRefill = now
	}

	// Check if we have tokens
	if bucket.tokens <= 0 {
		return false
	}

	bucket.tokens--
	return true
}

// cleanupExpiredTokens periodically cleans up expired CSRF tokens
func (sm *SecurityMiddleware) cleanupExpiredTokens() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for token, csrfToken := range sm.csrfTokens {
			if now.After(csrfToken.ExpiresAt) {
				delete(sm.csrfTokens, token)
			}
		}
		sm.mu.Unlock()
	}
}

// logSecurityEvent logs a security audit event
func (sm *SecurityMiddleware) logSecurityEvent(eventType logger.AuditEventType, description string, event string, err error) {
	if sm.auditLogger == nil {
		return
	}

	details := map[string]interface{}{
		"ipc_event": event,
	}

	if err != nil {
		sm.auditLogger.LogFailure(eventType, description, err, details)
	} else {
		sm.auditLogger.LogSuccess(eventType, description, details)
	}
}

// SensitiveEvents lists events that should be treated with extra care
var SensitiveEvents = map[string]bool{
	"config:save":    true,
	"node:start":     true,
	"node:stop":      true,
	"peers:add":      true,
	"peers:remove":   true,
	"settings:set":   true,
	"proxy:config":   true,
	"mapping:add":    true,
	"mapping:remove": true,
}

// IsSensitiveEvent checks if an event is sensitive
func IsSensitiveEvent(event string) bool {
	return SensitiveEvents[event]
}

// ValidationMiddleware validates event-specific payloads
func ValidationMiddleware(validator *security.Validator, log *logger.Logger) Middleware {
	return func(event string, req *Request, next func() *Response) *Response {
		// Event-specific validation
		switch event {
		case "peers:add":
			var payload struct {
				URI string `json:"uri"`
			}
			if err := parsePayload(req.Payload, &payload); err == nil {
				if err := validator.ValidatePeerURI(payload.URI); err != nil {
					log.Warn("Invalid peer URI", "event", event, "error", err)
					return &Response{
						Success: false,
						Error: &Error{
							Code:    "VALIDATION_ERROR",
							Message: err.Error(),
						},
					}
				}
			}

		case "proxy:config":
			var payload struct {
				ListenAddress string `json:"listenAddress"`
			}
			if err := parsePayload(req.Payload, &payload); err == nil && payload.ListenAddress != "" {
				if err := validator.ValidateListenAddress(payload.ListenAddress); err != nil {
					log.Warn("Invalid listen address", "event", event, "error", err)
					return &Response{
						Success: false,
						Error: &Error{
							Code:    "VALIDATION_ERROR",
							Message: err.Error(),
						},
					}
				}
			}

		case "settings:set":
			var payload map[string]interface{}
			if err := parsePayload(req.Payload, &payload); err == nil {
				if lang, ok := payload["language"].(string); ok {
					if err := validator.ValidateLanguage(lang); err != nil {
						log.Warn("Invalid language", "event", event, "error", err)
						return &Response{
							Success: false,
							Error: &Error{
								Code:    "VALIDATION_ERROR",
								Message: err.Error(),
							},
						}
					}
				}
				if theme, ok := payload["theme"].(string); ok {
					if err := validator.ValidateTheme(theme); err != nil {
						log.Warn("Invalid theme", "event", event, "error", err)
						return &Response{
							Success: false,
							Error: &Error{
								Code:    "VALIDATION_ERROR",
								Message: err.Error(),
							},
						}
					}
				}
				if level, ok := payload["logLevel"].(string); ok {
					if err := validator.ValidateLogLevel(level); err != nil {
						log.Warn("Invalid log level", "event", event, "error", err)
						return &Response{
							Success: false,
							Error: &Error{
								Code:    "VALIDATION_ERROR",
								Message: err.Error(),
							},
						}
					}
				}
			}
		}

		return next()
	}
}

// parsePayload is a helper to parse JSON payload
func parsePayload(data []byte, target interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
