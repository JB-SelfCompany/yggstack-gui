package security

import (
	"errors"
	"sync"
)

// Service name used for keychain operations
const (
	ServiceName = "yggstack-gui"
	AccountName = "yggdrasil-private-key"
)

// Common errors
var (
	ErrKeyNotFound     = errors.New("key not found in keychain")
	ErrKeychainAccess  = errors.New("failed to access keychain")
	ErrKeyTooLarge     = errors.New("key data exceeds maximum size")
	ErrInvalidKey      = errors.New("invalid key format")
	ErrKeychainLocked  = errors.New("keychain is locked")
	ErrNotImplemented  = errors.New("keychain not implemented for this platform")
	ErrFallbackFailed  = errors.New("fallback storage failed")
)

// Keychain defines the interface for secure credential storage
type Keychain interface {
	// Store saves a secret to the keychain
	Store(key string, value []byte) error

	// Retrieve gets a secret from the keychain
	Retrieve(key string) ([]byte, error)

	// Delete removes a secret from the keychain
	Delete(key string) error

	// IsAvailable checks if the keychain is accessible
	IsAvailable() bool
}

// SecureStore provides a unified interface for secure storage
// with automatic fallback to encrypted file storage
type SecureStore struct {
	mu       sync.RWMutex
	keychain Keychain
	fallback *EncryptedStore
	useFallback bool
}

// NewSecureStore creates a new secure storage with platform keychain
// and fallback to encrypted file storage
func NewSecureStore(fallbackPath string, machineKey []byte) (*SecureStore, error) {
	store := &SecureStore{}

	// Try to initialize platform keychain
	keychain := newPlatformKeychain()
	if keychain != nil && keychain.IsAvailable() {
		store.keychain = keychain
		store.useFallback = false
	} else {
		// Fall back to encrypted file storage
		var err error
		store.fallback, err = NewEncryptedStore(fallbackPath, machineKey)
		if err != nil {
			return nil, err
		}
		store.useFallback = true
	}

	return store, nil
}

// Store saves a secret securely
func (s *SecureStore) Store(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Zero memory after use
	defer ZeroBytes(value)

	if s.useFallback {
		return s.fallback.Store(key, value)
	}

	err := s.keychain.Store(key, value)
	if err != nil {
		// Try fallback if keychain fails
		if s.fallback != nil {
			return s.fallback.Store(key, value)
		}
		return err
	}

	return nil
}

// Retrieve gets a secret securely
func (s *SecureStore) Retrieve(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.useFallback {
		return s.fallback.Retrieve(key)
	}

	value, err := s.keychain.Retrieve(key)
	if err != nil {
		// Try fallback if keychain fails
		if s.fallback != nil {
			return s.fallback.Retrieve(key)
		}
		return nil, err
	}

	return value, nil
}

// Delete removes a secret
func (s *SecureStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.useFallback {
		return s.fallback.Delete(key)
	}

	err := s.keychain.Delete(key)
	if err != nil && s.fallback != nil {
		// Also try to delete from fallback
		_ = s.fallback.Delete(key)
	}

	return err
}

// IsUsingFallback returns true if using encrypted file storage
func (s *SecureStore) IsUsingFallback() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.useFallback
}

// Close closes the secure store and zeros any sensitive data
func (s *SecureStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fallback != nil {
		return s.fallback.Close()
	}

	return nil
}

// ZeroBytes securely zeros a byte slice
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// ZeroString securely zeros a string's underlying bytes
// Note: This only works if the string was created from a modifiable source
func ZeroString(s *string) {
	if s == nil || len(*s) == 0 {
		return
	}
	// Convert to byte slice for zeroing
	// This creates a copy, so the original string bytes may still be in memory
	// For truly secure handling, use []byte throughout
	b := []byte(*s)
	ZeroBytes(b)
	*s = ""
}

// SecureBytes wraps a byte slice and zeros it when done
type SecureBytes struct {
	data []byte
}

// NewSecureBytes creates a new SecureBytes wrapper
func NewSecureBytes(data []byte) *SecureBytes {
	return &SecureBytes{data: data}
}

// Data returns the underlying byte slice
func (s *SecureBytes) Data() []byte {
	return s.data
}

// Zero clears the underlying data
func (s *SecureBytes) Zero() {
	ZeroBytes(s.data)
	s.data = nil
}

// Len returns the length of the data
func (s *SecureBytes) Len() int {
	return len(s.data)
}
