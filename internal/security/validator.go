package security

import (
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ValidationError represents a validation error with context
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validator provides input validation utilities
type Validator struct {
	maxPeerURILength     int
	maxPortMappingLength int
	maxHostLength        int
	maxPathLength        int
}

// NewValidator creates a new validator with default limits
func NewValidator() *Validator {
	return &Validator{
		maxPeerURILength:     2048,
		maxPortMappingLength: 512,
		maxHostLength:        253,
		maxPathLength:        4096,
	}
}

// ValidatePeerURI validates a Yggdrasil peer URI with enhanced security checks
func (v *Validator) ValidatePeerURI(uri string) error {
	// Length check
	if len(uri) > v.maxPeerURILength {
		return &ValidationError{
			Field:   "uri",
			Message: fmt.Sprintf("URI exceeds maximum length of %d", v.maxPeerURILength),
			Code:    "URI_TOO_LONG",
		}
	}

	// Empty check
	if uri == "" {
		return &ValidationError{
			Field:   "uri",
			Message: "URI cannot be empty",
			Code:    "URI_EMPTY",
		}
	}

	// Check for dangerous characters
	if err := v.checkDangerousChars(uri, "uri"); err != nil {
		return err
	}

	// Parse URL
	u, err := url.Parse(uri)
	if err != nil {
		return &ValidationError{
			Field:   "uri",
			Message: fmt.Sprintf("Invalid URI format: %v", err),
			Code:    "URI_INVALID",
		}
	}

	// Validate scheme
	switch u.Scheme {
	case "tcp", "tls", "quic", "ws", "wss":
		// Valid schemes
	case "":
		return &ValidationError{
			Field:   "uri",
			Message: "URI must include a scheme (tcp, tls, quic, ws, or wss)",
			Code:    "SCHEME_MISSING",
		}
	default:
		return &ValidationError{
			Field:   "uri",
			Message: fmt.Sprintf("Unsupported scheme: %s (expected tcp, tls, quic, ws, or wss)", u.Scheme),
			Code:    "SCHEME_INVALID",
		}
	}

	// Validate host
	if u.Host == "" {
		return &ValidationError{
			Field:   "uri",
			Message: "URI must include a host",
			Code:    "HOST_MISSING",
		}
	}

	// Validate host:port
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		// Try without port
		host = u.Host
		portStr = ""
	}

	// Validate host format
	if err := v.ValidateHost(host); err != nil {
		return err
	}

	// Port is required for tcp, tls, quic schemes
	if portStr == "" {
		switch u.Scheme {
		case "tcp", "tls", "quic":
			return &ValidationError{
				Field:   "uri",
				Message: fmt.Sprintf("Port is required for %s:// URIs", u.Scheme),
				Code:    "PORT_MISSING",
			}
		}
	} else {
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			return &ValidationError{
				Field:   "uri",
				Message: fmt.Sprintf("Invalid port number: %s", portStr),
				Code:    "PORT_INVALID",
			}
		}
	}

	// Check for public key in query params (optional)
	if key := u.Query().Get("key"); key != "" {
		if err := v.ValidatePublicKey(key); err != nil {
			return err
		}
	}

	return nil
}

// ValidateHost validates a hostname or IP address
func (v *Validator) ValidateHost(host string) error {
	if host == "" {
		return &ValidationError{
			Field:   "host",
			Message: "Host cannot be empty",
			Code:    "HOST_EMPTY",
		}
	}

	if len(host) > v.maxHostLength {
		return &ValidationError{
			Field:   "host",
			Message: fmt.Sprintf("Host exceeds maximum length of %d", v.maxHostLength),
			Code:    "HOST_TOO_LONG",
		}
	}

	// Check if it's an IPv6 address
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		ipv6 := host[1 : len(host)-1]
		if ip := net.ParseIP(ipv6); ip == nil || ip.To4() != nil {
			return &ValidationError{
				Field:   "host",
				Message: "Invalid IPv6 address",
				Code:    "IPV6_INVALID",
			}
		}
		return nil
	}

	// Check if it's an IP address
	if ip := net.ParseIP(host); ip != nil {
		return nil
	}

	// Validate as hostname
	return v.validateHostname(host)
}

// validateHostname checks if a hostname is valid according to RFC 1123
func (v *Validator) validateHostname(hostname string) error {
	// Check for valid hostname pattern
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

	if !hostnameRegex.MatchString(hostname) {
		return &ValidationError{
			Field:   "host",
			Message: "Invalid hostname format",
			Code:    "HOSTNAME_INVALID",
		}
	}

	return nil
}

// ValidatePublicKey validates a Yggdrasil public key (hex-encoded ed25519 key)
func (v *Validator) ValidatePublicKey(key string) error {
	if key == "" {
		return &ValidationError{
			Field:   "publicKey",
			Message: "Public key cannot be empty",
			Code:    "PUBKEY_EMPTY",
		}
	}

	// Remove any prefix
	key = strings.TrimPrefix(key, "0x")

	// Check length (ed25519 public key is 32 bytes = 64 hex chars)
	if len(key) != 64 {
		return &ValidationError{
			Field:   "publicKey",
			Message: fmt.Sprintf("Public key must be 64 hex characters, got %d", len(key)),
			Code:    "PUBKEY_LENGTH",
		}
	}

	// Check if valid hex
	if _, err := hex.DecodeString(key); err != nil {
		return &ValidationError{
			Field:   "publicKey",
			Message: "Public key must be valid hexadecimal",
			Code:    "PUBKEY_HEX",
		}
	}

	return nil
}

// ValidateIPv6Address validates an Yggdrasil IPv6 address
func (v *Validator) ValidateIPv6Address(addr string) error {
	if addr == "" {
		return &ValidationError{
			Field:   "address",
			Message: "IPv6 address cannot be empty",
			Code:    "ADDR_EMPTY",
		}
	}

	ip := net.ParseIP(addr)
	if ip == nil {
		return &ValidationError{
			Field:   "address",
			Message: "Invalid IPv6 address format",
			Code:    "ADDR_INVALID",
		}
	}

	// Check if it's actually IPv6
	if ip.To4() != nil {
		return &ValidationError{
			Field:   "address",
			Message: "Expected IPv6 address, got IPv4",
			Code:    "ADDR_NOT_IPV6",
		}
	}

	// Check Yggdrasil prefix (0200::/7 or 0300::/7)
	if !strings.HasPrefix(addr, "2") && !strings.HasPrefix(addr, "3") {
		return &ValidationError{
			Field:   "address",
			Message: "Address does not appear to be a valid Yggdrasil address",
			Code:    "ADDR_NOT_YGGDRASIL",
		}
	}

	return nil
}

// ValidatePortMapping validates a port mapping configuration
func (v *Validator) ValidatePortMapping(mapping string) error {
	if len(mapping) > v.maxPortMappingLength {
		return &ValidationError{
			Field:   "mapping",
			Message: fmt.Sprintf("Mapping exceeds maximum length of %d", v.maxPortMappingLength),
			Code:    "MAPPING_TOO_LONG",
		}
	}

	if mapping == "" {
		return &ValidationError{
			Field:   "mapping",
			Message: "Mapping cannot be empty",
			Code:    "MAPPING_EMPTY",
		}
	}

	// Check for dangerous characters
	if err := v.checkDangerousChars(mapping, "mapping"); err != nil {
		return err
	}

	// Parse mapping format: [local_address:]local_port:remote_address:remote_port
	parts := strings.Split(mapping, ":")
	if len(parts) < 3 || len(parts) > 5 {
		return &ValidationError{
			Field:   "mapping",
			Message: "Invalid mapping format. Expected [local_address:]local_port:remote_address:remote_port",
			Code:    "MAPPING_FORMAT",
		}
	}

	return nil
}

// ValidatePort validates a port number
func (v *Validator) ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return &ValidationError{
			Field:   "port",
			Message: fmt.Sprintf("Port must be between 1 and 65535, got %d", port),
			Code:    "PORT_RANGE",
		}
	}

	return nil
}

// ValidateListenAddress validates a listen address (host:port)
func (v *Validator) ValidateListenAddress(addr string) error {
	if addr == "" {
		return &ValidationError{
			Field:   "listenAddress",
			Message: "Listen address cannot be empty",
			Code:    "LISTEN_EMPTY",
		}
	}

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return &ValidationError{
			Field:   "listenAddress",
			Message: fmt.Sprintf("Invalid address format: %v", err),
			Code:    "LISTEN_FORMAT",
		}
	}

	// Host can be empty (listen on all interfaces)
	if host != "" {
		if err := v.ValidateHost(host); err != nil {
			return err
		}
	}

	// Validate port
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return &ValidationError{
			Field:   "listenAddress",
			Message: "Invalid port number",
			Code:    "PORT_INVALID",
		}
	}

	return v.ValidatePort(port)
}

// checkDangerousChars checks for potentially dangerous characters
func (v *Validator) checkDangerousChars(input string, field string) error {
	// Check for null bytes
	if strings.Contains(input, "\x00") {
		return &ValidationError{
			Field:   field,
			Message: "Input contains null bytes",
			Code:    "NULL_BYTES",
		}
	}

	// Check for control characters (except allowed ones)
	for _, r := range input {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return &ValidationError{
				Field:   field,
				Message: "Input contains invalid control characters",
				Code:    "CONTROL_CHARS",
			}
		}
	}

	// Check for shell injection patterns
	shellPatterns := []string{
		"$(", "`", "${", "||", "&&", ";", "|", ">", "<", "&",
	}

	for _, pattern := range shellPatterns {
		if strings.Contains(input, pattern) {
			return &ValidationError{
				Field:   field,
				Message: "Input contains potentially dangerous characters",
				Code:    "DANGEROUS_CHARS",
			}
		}
	}

	return nil
}

// SanitizeLogString removes or redacts sensitive information from a string for logging
func (v *Validator) SanitizeLogString(input string) string {
	// Redact potential passwords
	passwordPatterns := []string{
		"password=", "passwd=", "pwd=", "secret=", "key=", "token=",
	}

	result := input
	for _, pattern := range passwordPatterns {
		if idx := strings.Index(strings.ToLower(result), pattern); idx >= 0 {
			endIdx := strings.IndexAny(result[idx+len(pattern):], "&? \n\r\t")
			if endIdx == -1 {
				endIdx = len(result) - idx - len(pattern)
			}
			result = result[:idx+len(pattern)] + "[REDACTED]" + result[idx+len(pattern)+endIdx:]
		}
	}

	// Redact public keys (64 hex chars)
	keyRegex := regexp.MustCompile(`[0-9a-fA-F]{64}`)
	result = keyRegex.ReplaceAllString(result, "[KEY_REDACTED]")

	return result
}

// ValidateJSON validates that a string is valid JSON and doesn't exceed size limits
func (v *Validator) ValidateJSON(input string, maxSize int) error {
	if len(input) > maxSize {
		return &ValidationError{
			Field:   "json",
			Message: fmt.Sprintf("JSON exceeds maximum size of %d bytes", maxSize),
			Code:    "JSON_TOO_LARGE",
		}
	}

	// Check for null bytes
	if strings.Contains(input, "\x00") {
		return &ValidationError{
			Field:   "json",
			Message: "JSON contains null bytes",
			Code:    "JSON_NULL_BYTES",
		}
	}

	return nil
}

// ValidateLanguage validates a language code
func (v *Validator) ValidateLanguage(lang string) error {
	validLanguages := map[string]bool{
		"en": true,
		"ru": true,
	}

	if !validLanguages[lang] {
		return &ValidationError{
			Field:   "language",
			Message: fmt.Sprintf("Unsupported language: %s (expected en or ru)", lang),
			Code:    "LANG_INVALID",
		}
	}

	return nil
}

// ValidateTheme validates a theme name
func (v *Validator) ValidateTheme(theme string) error {
	validThemes := map[string]bool{
		"light":  true,
		"dark":   true,
		"system": true,
	}

	if !validThemes[theme] {
		return &ValidationError{
			Field:   "theme",
			Message: fmt.Sprintf("Unsupported theme: %s (expected light, dark, or system)", theme),
			Code:    "THEME_INVALID",
		}
	}

	return nil
}

// ValidateLogLevel validates a log level
func (v *Validator) ValidateLogLevel(level string) error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLevels[level] {
		return &ValidationError{
			Field:   "logLevel",
			Message: fmt.Sprintf("Unsupported log level: %s", level),
			Code:    "LEVEL_INVALID",
		}
	}

	return nil
}
