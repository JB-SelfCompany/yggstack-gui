package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// AES-256 key size
	keySize = 32
	// PBKDF2 iterations
	pbkdf2Iterations = 100000
	// Salt size
	saltSize = 32
	// Nonce size for GCM
	nonceSize = 12
)

var (
	ErrDecryptionFailed = errors.New("decryption failed: invalid key or corrupted data")
	ErrEncryptionFailed = errors.New("encryption failed")
)

// EncryptedStore provides AES-256-GCM encrypted storage
type EncryptedStore struct {
	mu       sync.RWMutex
	path     string
	key      []byte
	salt     []byte
	data     map[string][]byte
	modified bool
}

// encryptedFile represents the on-disk format
type encryptedFile struct {
	Version int    `json:"version"`
	Salt    []byte `json:"salt"`
	Data    []byte `json:"data"` // Encrypted JSON of map[string][]byte
}

// NewEncryptedStore creates a new encrypted store
// machineKey should be derived from machine-specific data for added security
func NewEncryptedStore(path string, machineKey []byte) (*EncryptedStore, error) {
	store := &EncryptedStore{
		path: path,
		data: make(map[string][]byte),
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Generate new salt
		store.salt = make([]byte, saltSize)
		if _, err := io.ReadFull(rand.Reader, store.salt); err != nil {
			return nil, err
		}
		// Derive key from machine key
		store.key = deriveKey(machineKey, store.salt)
		return store, nil
	}

	// Load existing file
	if err := store.load(machineKey); err != nil {
		return nil, err
	}

	return store, nil
}

// deriveKey derives an AES-256 key using PBKDF2
func deriveKey(password, salt []byte) []byte {
	return pbkdf2.Key(password, salt, pbkdf2Iterations, keySize, sha256.New)
}

// load reads and decrypts the store from disk
func (s *EncryptedStore) load(machineKey []byte) error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	var ef encryptedFile
	if err := json.Unmarshal(data, &ef); err != nil {
		return err
	}

	s.salt = ef.Salt
	s.key = deriveKey(machineKey, s.salt)

	// Decrypt the data
	plaintext, err := decrypt(ef.Data, s.key)
	if err != nil {
		return ErrDecryptionFailed
	}

	// Parse the decrypted data
	if err := json.Unmarshal(plaintext, &s.data); err != nil {
		ZeroBytes(plaintext)
		return err
	}

	ZeroBytes(plaintext)
	return nil
}

// save encrypts and writes the store to disk
func (s *EncryptedStore) save() error {
	// Serialize the data
	plaintext, err := json.Marshal(s.data)
	if err != nil {
		return err
	}
	defer ZeroBytes(plaintext)

	// Encrypt
	ciphertext, err := encrypt(plaintext, s.key)
	if err != nil {
		return err
	}

	// Create the file structure
	ef := encryptedFile{
		Version: 1,
		Salt:    s.salt,
		Data:    ciphertext,
	}

	fileData, err := json.Marshal(ef)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Write atomically
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, fileData, 0600); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.path)
}

// Store saves a value to the encrypted store
func (s *EncryptedStore) Store(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Make a copy of the value
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	s.data[key] = valueCopy
	s.modified = true

	return s.save()
}

// Retrieve gets a value from the encrypted store
func (s *EncryptedStore) Retrieve(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.data[key]
	if !ok {
		return nil, ErrKeyNotFound
	}

	// Return a copy
	result := make([]byte, len(value))
	copy(result, value)

	return result, nil
}

// Delete removes a value from the encrypted store
func (s *EncryptedStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if value, ok := s.data[key]; ok {
		ZeroBytes(value)
		delete(s.data, key)
		s.modified = true
		return s.save()
	}

	return nil
}

// Close securely zeros all sensitive data
func (s *EncryptedStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Zero all stored values
	for key, value := range s.data {
		ZeroBytes(value)
		delete(s.data, key)
	}

	// Zero the key
	ZeroBytes(s.key)

	return nil
}

// encrypt encrypts data using AES-256-GCM
func encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// nonce is prepended to the ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// decrypt decrypts data using AES-256-GCM
func decrypt(ciphertext, key []byte) ([]byte, error) {
	if len(ciphertext) < nonceSize {
		return nil, ErrDecryptionFailed
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// GenerateRandomBytes generates cryptographically secure random bytes
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, err
	}
	return b, nil
}

// GenerateMachineKey generates a machine-specific key
// This combines various system identifiers to create a unique key
func GenerateMachineKey() ([]byte, error) {
	// Start with random bytes as base
	randomPart, err := GenerateRandomBytes(16)
	if err != nil {
		return nil, err
	}

	// Combine with machine-specific data
	// The hash ensures consistent length regardless of input
	h := sha256.New()
	h.Write(randomPart)

	// Add machine-specific identifiers
	// These are read at runtime for each platform
	machineID := getMachineID()
	h.Write([]byte(machineID))

	return h.Sum(nil), nil
}

// getMachineID returns a platform-specific machine identifier
func getMachineID() string {
	// Try common locations for machine ID
	locations := []string{
		"/etc/machine-id",           // Linux
		"/var/lib/dbus/machine-id",  // Linux fallback
	}

	for _, loc := range locations {
		if data, err := os.ReadFile(loc); err == nil {
			return string(data)
		}
	}

	// Fallback: use hostname + current user
	hostname, _ := os.Hostname()
	homeDir, _ := os.UserHomeDir()

	h := sha256.New()
	h.Write([]byte(hostname))
	h.Write([]byte(homeDir))

	return string(h.Sum(nil))
}

// DeriveKeyFromPassword derives an encryption key from a user password
func DeriveKeyFromPassword(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, keySize, sha256.New)
}

// HashPassword hashes a password using SHA-256
// For password storage, use bcrypt or argon2 instead
func HashPassword(password string) []byte {
	h := sha256.Sum256([]byte(password))
	return h[:]
}

// ConstantTimeCompare performs constant-time comparison of two byte slices
// to prevent timing attacks
func ConstantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}

	return result == 0
}
