//go:build !windows && !darwin && !linux

package security

// FallbackKeychain is used on unsupported platforms
type FallbackKeychain struct{}

// newPlatformKeychain creates a new platform-specific keychain
// On unsupported platforms, returns nil to trigger fallback to encrypted storage
func newPlatformKeychain() Keychain {
	return nil
}

// Store is not implemented on this platform
func (k *FallbackKeychain) Store(key string, value []byte) error {
	return ErrNotImplemented
}

// Retrieve is not implemented on this platform
func (k *FallbackKeychain) Retrieve(key string) ([]byte, error) {
	return nil, ErrNotImplemented
}

// Delete is not implemented on this platform
func (k *FallbackKeychain) Delete(key string) error {
	return ErrNotImplemented
}

// IsAvailable returns false on unsupported platforms
func (k *FallbackKeychain) IsAvailable() bool {
	return false
}
