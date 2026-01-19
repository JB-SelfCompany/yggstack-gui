//go:build darwin

package security

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"os/exec"
	"strings"
)

// DarwinKeychain implements the Keychain interface using macOS Keychain
type DarwinKeychain struct {
	service string
}

// newPlatformKeychain creates a new platform-specific keychain
func newPlatformKeychain() Keychain {
	return &DarwinKeychain{
		service: ServiceName,
	}
}

// Store saves a secret to macOS Keychain
func (k *DarwinKeychain) Store(key string, value []byte) error {
	// First, try to delete existing entry (ignore errors)
	_ = k.Delete(key)

	// Encode value as base64 for safe storage
	encoded := base64.StdEncoding.EncodeToString(value)

	cmd := exec.Command(
		"security",
		"add-generic-password",
		"-s", k.service,
		"-a", key,
		"-w", encoded,
		"-U", // Update if exists
	)

	if err := cmd.Run(); err != nil {
		return ErrKeychainAccess
	}

	return nil
}

// Retrieve gets a secret from macOS Keychain
func (k *DarwinKeychain) Retrieve(key string) ([]byte, error) {
	cmd := exec.Command(
		"security",
		"find-generic-password",
		"-s", k.service,
		"-a", key,
		"-w", // Output password only
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check if it's a "not found" error
		if strings.Contains(stderr.String(), "could not be found") ||
			strings.Contains(stderr.String(), "SecKeychainSearchCopyNext") {
			return nil, ErrKeyNotFound
		}
		return nil, ErrKeychainAccess
	}

	// Decode from base64
	encoded := strings.TrimSpace(stdout.String())
	value, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		// Try as hex (legacy format)
		value, err = hex.DecodeString(encoded)
		if err != nil {
			// Return as-is if not encoded
			return []byte(encoded), nil
		}
	}

	return value, nil
}

// Delete removes a secret from macOS Keychain
func (k *DarwinKeychain) Delete(key string) error {
	cmd := exec.Command(
		"security",
		"delete-generic-password",
		"-s", k.service,
		"-a", key,
	)

	if err := cmd.Run(); err != nil {
		return ErrKeyNotFound
	}

	return nil
}

// IsAvailable checks if macOS Keychain is accessible
func (k *DarwinKeychain) IsAvailable() bool {
	// Check if security command is available
	cmd := exec.Command("security", "list-keychains")
	err := cmd.Run()
	return err == nil
}
