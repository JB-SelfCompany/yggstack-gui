//go:build windows

package security

import (
	"syscall"
	"unsafe"
)

var (
	modadvapi32         = syscall.NewLazyDLL("advapi32.dll")
	procCredWriteW      = modadvapi32.NewProc("CredWriteW")
	procCredReadW       = modadvapi32.NewProc("CredReadW")
	procCredDeleteW     = modadvapi32.NewProc("CredDeleteW")
	procCredFree        = modadvapi32.NewProc("CredFree")
)

const (
	CRED_TYPE_GENERIC          = 1
	CRED_PERSIST_LOCAL_MACHINE = 2
)

// CREDENTIAL structure for Windows Credential Manager
type credential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        syscall.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

// WindowsKeychain implements the Keychain interface using Windows Credential Manager
type WindowsKeychain struct {
	service string
}

// newPlatformKeychain creates a new platform-specific keychain
func newPlatformKeychain() Keychain {
	return &WindowsKeychain{
		service: ServiceName,
	}
}

// buildTargetName creates the target name for the credential
func (k *WindowsKeychain) buildTargetName(key string) string {
	return k.service + ":" + key
}

// Store saves a secret to Windows Credential Manager
func (k *WindowsKeychain) Store(key string, value []byte) error {
	targetName, err := syscall.UTF16PtrFromString(k.buildTargetName(key))
	if err != nil {
		return err
	}

	userName, err := syscall.UTF16PtrFromString(key)
	if err != nil {
		return err
	}

	cred := credential{
		Type:               CRED_TYPE_GENERIC,
		TargetName:         targetName,
		CredentialBlobSize: uint32(len(value)),
		CredentialBlob:     &value[0],
		Persist:            CRED_PERSIST_LOCAL_MACHINE,
		UserName:           userName,
	}

	ret, _, err := procCredWriteW.Call(
		uintptr(unsafe.Pointer(&cred)),
		0,
	)

	if ret == 0 {
		return ErrKeychainAccess
	}

	return nil
}

// Retrieve gets a secret from Windows Credential Manager
func (k *WindowsKeychain) Retrieve(key string) ([]byte, error) {
	targetName, err := syscall.UTF16PtrFromString(k.buildTargetName(key))
	if err != nil {
		return nil, err
	}

	var pcred *credential
	ret, _, err := procCredReadW.Call(
		uintptr(unsafe.Pointer(targetName)),
		CRED_TYPE_GENERIC,
		0,
		uintptr(unsafe.Pointer(&pcred)),
	)

	if ret == 0 {
		return nil, ErrKeyNotFound
	}

	defer procCredFree.Call(uintptr(unsafe.Pointer(pcred)))

	// Copy the credential blob
	size := pcred.CredentialBlobSize
	if size == 0 {
		return nil, ErrKeyNotFound
	}

	result := make([]byte, size)
	src := unsafe.Slice(pcred.CredentialBlob, size)
	copy(result, src)

	return result, nil
}

// Delete removes a secret from Windows Credential Manager
func (k *WindowsKeychain) Delete(key string) error {
	targetName, err := syscall.UTF16PtrFromString(k.buildTargetName(key))
	if err != nil {
		return err
	}

	ret, _, _ := procCredDeleteW.Call(
		uintptr(unsafe.Pointer(targetName)),
		CRED_TYPE_GENERIC,
		0,
	)

	if ret == 0 {
		return ErrKeyNotFound
	}

	return nil
}

// IsAvailable checks if Windows Credential Manager is accessible
func (k *WindowsKeychain) IsAvailable() bool {
	// Windows Credential Manager is always available on Windows
	err := modadvapi32.Load()
	return err == nil
}
