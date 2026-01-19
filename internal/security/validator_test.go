package security

import (
	"strings"
	"testing"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("NewValidator returned nil")
	}
	if v.maxPeerURILength != 2048 {
		t.Errorf("expected maxPeerURILength 2048, got %d", v.maxPeerURILength)
	}
	if v.maxHostLength != 253 {
		t.Errorf("expected maxHostLength 253, got %d", v.maxHostLength)
	}
}

func TestValidatePeerURI(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		uri     string
		wantErr bool
		errCode string
	}{
		// Valid URIs
		{
			name:    "valid tcp uri",
			uri:     "tcp://example.com:12345",
			wantErr: false,
		},
		{
			name:    "valid tls uri",
			uri:     "tls://peer.yggdrasil.io:443",
			wantErr: false,
		},
		{
			name:    "valid quic uri",
			uri:     "quic://node.example.org:9001",
			wantErr: false,
		},
		{
			name:    "valid ws uri",
			uri:     "ws://websocket.example.com:8080",
			wantErr: false,
		},
		{
			name:    "valid wss uri",
			uri:     "wss://secure.example.com:443",
			wantErr: false,
		},
		{
			name:    "valid tcp with ipv4",
			uri:     "tcp://192.168.1.1:12345",
			wantErr: false,
		},
		{
			name:    "valid tcp with ipv6",
			uri:     "tcp://[2001:db8::1]:12345",
			wantErr: false,
		},
		{
			name:    "valid uri with public key",
			uri:     "tls://example.com:443?key=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			wantErr: false,
		},

		// Invalid URIs
		{
			name:    "empty uri",
			uri:     "",
			wantErr: true,
			errCode: "URI_EMPTY",
		},
		{
			name:    "missing scheme",
			uri:     "example.com:12345",
			wantErr: true,
			errCode: "SCHEME_MISSING",
		},
		{
			name:    "invalid scheme",
			uri:     "http://example.com:12345",
			wantErr: true,
			errCode: "SCHEME_INVALID",
		},
		{
			name:    "missing host",
			uri:     "tcp://:12345",
			wantErr: true,
			errCode: "HOST_EMPTY",
		},
		{
			name:    "invalid port - too high",
			uri:     "tcp://example.com:99999",
			wantErr: true,
			errCode: "PORT_INVALID",
		},
		{
			name:    "invalid port - negative",
			uri:     "tcp://example.com:-1",
			wantErr: true,
			errCode: "PORT_INVALID",
		},
		{
			name:    "invalid port - zero",
			uri:     "tcp://example.com:0",
			wantErr: true,
			errCode: "PORT_INVALID",
		},
		{
			name:    "uri too long",
			uri:     "tcp://" + strings.Repeat("a", 2050) + ".com:12345",
			wantErr: true,
			errCode: "URI_TOO_LONG",
		},
		{
			name:    "dangerous chars - shell injection",
			uri:     "tcp://example.com:12345;rm -rf /",
			wantErr: true,
			errCode: "DANGEROUS_CHARS",
		},
		{
			name:    "dangerous chars - command substitution",
			uri:     "tcp://$(whoami).com:12345",
			wantErr: true,
			errCode: "DANGEROUS_CHARS",
		},
		{
			name:    "null byte injection",
			uri:     "tcp://example\x00.com:12345",
			wantErr: true,
			errCode: "NULL_BYTES",
		},
		{
			name:    "invalid public key - wrong length",
			uri:     "tls://example.com:443?key=0123456789abcdef",
			wantErr: true,
			errCode: "PUBKEY_LENGTH",
		},
		{
			name:    "invalid public key - not hex",
			uri:     "tls://example.com:443?key=ghijklmnopqrstuv0123456789abcdef0123456789abcdef0123456789abcdef",
			wantErr: true,
			errCode: "PUBKEY_HEX",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidatePeerURI(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePeerURI(%q) expected error, got nil", tt.uri)
					return
				}
				if ve, ok := err.(*ValidationError); ok {
					if tt.errCode != "" && ve.Code != tt.errCode {
						t.Errorf("ValidatePeerURI(%q) expected error code %q, got %q", tt.uri, tt.errCode, ve.Code)
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePeerURI(%q) unexpected error: %v", tt.uri, err)
				}
			}
		})
	}
}

func TestValidateHost(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		host    string
		wantErr bool
		errCode string
	}{
		// Valid hosts
		{"valid hostname", "example.com", false, ""},
		{"valid subdomain", "sub.example.com", false, ""},
		{"valid ipv4", "192.168.1.1", false, ""},
		{"valid ipv6 brackets", "[2001:db8::1]", false, ""},
		{"valid localhost", "localhost", false, ""},
		{"valid with numbers", "server1.example.com", false, ""},
		{"valid with hyphens", "my-server.example.com", false, ""},

		// Invalid hosts
		{"empty host", "", true, "HOST_EMPTY"},
		{"host too long", strings.Repeat("a", 254), true, "HOST_TOO_LONG"},
		{"invalid ipv6 brackets", "[not-ipv6]", true, "IPV6_INVALID"},
		{"hostname starts with hyphen", "-example.com", true, "HOSTNAME_INVALID"},
		{"hostname ends with hyphen", "example-.com", true, "HOSTNAME_INVALID"},
		{"hostname with underscore", "my_server.com", true, "HOSTNAME_INVALID"},
		{"hostname with space", "my server.com", true, "HOSTNAME_INVALID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateHost(tt.host)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateHost(%q) expected error, got nil", tt.host)
					return
				}
				if ve, ok := err.(*ValidationError); ok && tt.errCode != "" {
					if ve.Code != tt.errCode {
						t.Errorf("ValidateHost(%q) expected code %q, got %q", tt.host, tt.errCode, ve.Code)
					}
				}
			} else if err != nil {
				t.Errorf("ValidateHost(%q) unexpected error: %v", tt.host, err)
			}
		})
	}
}

func TestValidatePublicKey(t *testing.T) {
	v := NewValidator()

	validKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name    string
		key     string
		wantErr bool
		errCode string
	}{
		{"valid key", validKey, false, ""},
		{"valid key with 0x prefix", "0x" + validKey, false, ""},
		{"valid key uppercase", strings.ToUpper(validKey), false, ""},
		{"empty key", "", true, "PUBKEY_EMPTY"},
		{"key too short", "0123456789abcdef", true, "PUBKEY_LENGTH"},
		{"key too long", validKey + "00", true, "PUBKEY_LENGTH"},
		{"key not hex", strings.Repeat("g", 64), true, "PUBKEY_HEX"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidatePublicKey(tt.key)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePublicKey(%q) expected error", tt.key)
					return
				}
				if ve, ok := err.(*ValidationError); ok && tt.errCode != "" {
					if ve.Code != tt.errCode {
						t.Errorf("expected code %q, got %q", tt.errCode, ve.Code)
					}
				}
			} else if err != nil {
				t.Errorf("ValidatePublicKey(%q) unexpected error: %v", tt.key, err)
			}
		})
	}
}

func TestValidateIPv6Address(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
		errCode string
	}{
		// Valid Yggdrasil addresses (start with 2 or 3)
		{"valid yggdrasil addr 2xx", "200:1234:5678:9abc:def0:1234:5678:9abc", false, ""},
		{"valid yggdrasil addr 3xx", "300:abcd:ef01:2345:6789:abcd:ef01:2345", false, ""},

		// Invalid addresses
		{"empty address", "", true, "ADDR_EMPTY"},
		{"invalid format", "not-an-ip", true, "ADDR_INVALID"},
		{"ipv4 address", "192.168.1.1", true, "ADDR_NOT_IPV6"},
		{"non-yggdrasil ipv6", "fe80::1", true, "ADDR_NOT_YGGDRASIL"},
		{"non-yggdrasil ipv6 2", "2001:db8::1", true, "ADDR_NOT_YGGDRASIL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateIPv6Address(tt.addr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateIPv6Address(%q) expected error", tt.addr)
					return
				}
				if ve, ok := err.(*ValidationError); ok && tt.errCode != "" {
					if ve.Code != tt.errCode {
						t.Errorf("expected code %q, got %q", tt.errCode, ve.Code)
					}
				}
			} else if err != nil {
				t.Errorf("ValidateIPv6Address(%q) unexpected error: %v", tt.addr, err)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		port    int
		wantErr bool
	}{
		{1, false},
		{80, false},
		{443, false},
		{8080, false},
		{65535, false},
		{0, true},
		{-1, true},
		{65536, true},
		{100000, true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			err := v.ValidatePort(tt.port)
			if tt.wantErr && err == nil {
				t.Errorf("ValidatePort(%d) expected error", tt.port)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidatePort(%d) unexpected error: %v", tt.port, err)
			}
		})
	}
}

func TestValidateListenAddress(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{"valid localhost", "127.0.0.1:8080", false},
		{"valid all interfaces", ":8080", false},
		{"valid with hostname", "localhost:9000", false},
		{"valid ipv6", "[::1]:8080", false},

		{"empty", "", true},
		{"no port", "127.0.0.1", true},
		{"invalid port", "127.0.0.1:abc", true},
		{"port out of range", "127.0.0.1:99999", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateListenAddress(tt.addr)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateListenAddress(%q) expected error", tt.addr)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateListenAddress(%q) unexpected error: %v", tt.addr, err)
			}
		})
	}
}

func TestValidatePortMapping(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		mapping string
		wantErr bool
	}{
		{"valid 3 parts", "8080:192.168.1.1:80", false},
		{"valid 4 parts", "127.0.0.1:8080:192.168.1.1:80", false},

		{"empty", "", true},
		{"too few parts", "8080:80", true},
		{"too many parts", "a:b:c:d:e:f", true},
		{"too long", strings.Repeat("a", 513), true},
		{"shell injection", "8080;rm -rf:host:80", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidatePortMapping(tt.mapping)
			if tt.wantErr && err == nil {
				t.Errorf("ValidatePortMapping(%q) expected error", tt.mapping)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidatePortMapping(%q) unexpected error: %v", tt.mapping, err)
			}
		})
	}
}

func TestValidateLanguage(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		lang    string
		wantErr bool
	}{
		{"en", false},
		{"ru", false},
		{"", true},
		{"fr", true},
		{"EN", true}, // case sensitive
	}

	for _, tt := range tests {
		err := v.ValidateLanguage(tt.lang)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateLanguage(%q) expected error", tt.lang)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateLanguage(%q) unexpected error: %v", tt.lang, err)
		}
	}
}

func TestValidateTheme(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		theme   string
		wantErr bool
	}{
		{"light", false},
		{"dark", false},
		{"system", false},
		{"", true},
		{"blue", true},
		{"DARK", true},
	}

	for _, tt := range tests {
		err := v.ValidateTheme(tt.theme)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateTheme(%q) expected error", tt.theme)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateTheme(%q) unexpected error: %v", tt.theme, err)
		}
	}
}

func TestValidateLogLevel(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		level   string
		wantErr bool
	}{
		{"debug", false},
		{"info", false},
		{"warn", false},
		{"error", false},
		{"", true},
		{"trace", true},
		{"DEBUG", true},
	}

	for _, tt := range tests {
		err := v.ValidateLogLevel(tt.level)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateLogLevel(%q) expected error", tt.level)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateLogLevel(%q) unexpected error: %v", tt.level, err)
		}
	}
}

func TestSanitizeLogString(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name     string
		input    string
		contains string
		notContains string
	}{
		{
			name:        "redact password param",
			input:       "url?password=secret123&user=admin",
			contains:    "[REDACTED]",
			notContains: "secret123",
		},
		{
			name:        "redact key param",
			input:       "config?key=mysecretkey&foo=bar",
			contains:    "[REDACTED]",
			notContains: "mysecretkey",
		},
		{
			name:        "redact public key",
			input:       "peer key: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			contains:    "[KEY_REDACTED]",
			notContains: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		{
			name:     "no sensitive data",
			input:    "normal log message",
			contains: "normal log message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.SanitizeLogString(tt.input)
			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain %q, got %q", tt.contains, result)
			}
			if tt.notContains != "" && strings.Contains(result, tt.notContains) {
				t.Errorf("expected result NOT to contain %q, got %q", tt.notContains, result)
			}
		})
	}
}

func TestValidateJSON(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		input   string
		maxSize int
		wantErr bool
		errCode string
	}{
		{"valid small json", `{"key": "value"}`, 1000, false, ""},
		{"valid at limit", strings.Repeat("a", 100), 100, false, ""},
		{"too large", strings.Repeat("a", 101), 100, true, "JSON_TOO_LARGE"},
		{"contains null byte", "test\x00data", 1000, true, "JSON_NULL_BYTES"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateJSON(tt.input, tt.maxSize)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
					return
				}
				if ve, ok := err.(*ValidationError); ok && tt.errCode != "" {
					if ve.Code != tt.errCode {
						t.Errorf("expected code %q, got %q", tt.errCode, ve.Code)
					}
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "testField",
		Message: "test message",
		Code:    "TEST_CODE",
	}

	expected := "testField: test message"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

// Benchmark tests
func BenchmarkValidatePeerURI(b *testing.B) {
	v := NewValidator()
	uri := "tls://example.com:443?key=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ValidatePeerURI(uri)
	}
}

func BenchmarkValidateHost(b *testing.B) {
	v := NewValidator()
	host := "subdomain.example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ValidateHost(host)
	}
}

func BenchmarkSanitizeLogString(b *testing.B) {
	v := NewValidator()
	input := "password=secret&key=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.SanitizeLogString(input)
	}
}
