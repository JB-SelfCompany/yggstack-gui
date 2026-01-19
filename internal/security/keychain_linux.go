//go:build linux

package security

import (
	"bytes"
	"encoding/base64"
	"os/exec"
	"strings"
)

// LinuxKeychain implements the Keychain interface using Secret Service (libsecret)
type LinuxKeychain struct {
	service string
}

// newPlatformKeychain creates a new platform-specific keychain
func newPlatformKeychain() Keychain {
	return &LinuxKeychain{
		service: ServiceName,
	}
}

// Store saves a secret to Secret Service
func (k *LinuxKeychain) Store(key string, value []byte) error {
	// Encode value as base64 for safe storage
	encoded := base64.StdEncoding.EncodeToString(value)

	// Use secret-tool to store the password
	cmd := exec.Command(
		"secret-tool",
		"store",
		"--label", k.service+":"+key,
		"service", k.service,
		"account", key,
	)

	// Pass the password via stdin
	cmd.Stdin = strings.NewReader(encoded)

	if err := cmd.Run(); err != nil {
		// Try with pass as fallback
		return k.storeWithPass(key, encoded)
	}

	return nil
}

// storeWithPass uses pass (password-store) as fallback
func (k *LinuxKeychain) storeWithPass(key string, value string) error {
	passPath := k.service + "/" + key

	cmd := exec.Command("pass", "insert", "-f", passPath)
	cmd.Stdin = strings.NewReader(value + "\n" + value + "\n")

	if err := cmd.Run(); err != nil {
		return ErrKeychainAccess
	}

	return nil
}

// Retrieve gets a secret from Secret Service
func (k *LinuxKeychain) Retrieve(key string) ([]byte, error) {
	// Use secret-tool to lookup the password
	cmd := exec.Command(
		"secret-tool",
		"lookup",
		"service", k.service,
		"account", key,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Try with pass as fallback
		return k.retrieveWithPass(key)
	}

	// Decode from base64
	encoded := strings.TrimSpace(stdout.String())
	if encoded == "" {
		return nil, ErrKeyNotFound
	}

	value, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		// Return as-is if not encoded
		return []byte(encoded), nil
	}

	return value, nil
}

// retrieveWithPass uses pass (password-store) as fallback
func (k *LinuxKeychain) retrieveWithPass(key string) ([]byte, error) {
	passPath := k.service + "/" + key

	cmd := exec.Command("pass", "show", passPath)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, ErrKeyNotFound
	}

	encoded := strings.TrimSpace(stdout.String())
	value, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return []byte(encoded), nil
	}

	return value, nil
}

// Delete removes a secret from Secret Service
func (k *LinuxKeychain) Delete(key string) error {
	// Use secret-tool to clear the password
	cmd := exec.Command(
		"secret-tool",
		"clear",
		"service", k.service,
		"account", key,
	)

	if err := cmd.Run(); err != nil {
		// Try with pass as fallback
		return k.deleteWithPass(key)
	}

	return nil
}

// deleteWithPass uses pass (password-store) as fallback
func (k *LinuxKeychain) deleteWithPass(key string) error {
	passPath := k.service + "/" + key

	cmd := exec.Command("pass", "rm", "-f", passPath)

	if err := cmd.Run(); err != nil {
		return ErrKeyNotFound
	}

	return nil
}

// IsAvailable checks if Secret Service is accessible
func (k *LinuxKeychain) IsAvailable() bool {
	// Check if secret-tool is available
	cmd := exec.Command("secret-tool", "--version")
	if err := cmd.Run(); err == nil {
		return true
	}

	// Check if pass is available as fallback
	cmd = exec.Command("pass", "version")
	return cmd.Run() == nil
}
