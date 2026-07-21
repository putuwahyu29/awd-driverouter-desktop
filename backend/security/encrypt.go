//go:build windows

package security

import (
	"encoding/base64"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var crypt32 = windows.NewLazySystemDLL("crypt32.dll")
var procCryptProtectData = crypt32.NewProc("CryptProtectData")
var procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func newBlob(data []byte) *dataBlob {
	if len(data) == 0 {
		return &dataBlob{}
	}
	return &dataBlob{cbData: uint32(len(data)), pbData: &data[0]}
}

func (b *dataBlob) toByteSlice() []byte {
	if b.pbData == nil {
		return nil
	}
	return unsafe.Slice(b.pbData, b.cbData)
}

// Encrypt encrypts a plain-text string using Windows DPAPI and returns a base64-encoded result.
// The encryption is scoped to the current user account on this machine.
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	data := []byte(plaintext)
	input := newBlob(data)
	var output dataBlob

	r, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(input)),
		0,    // description
		0,    // optional entropy
		0,    // reserved
		0,    // prompt struct
		0,    // flags
		uintptr(unsafe.Pointer(&output)),
	)
	if r == 0 {
		return "", fmt.Errorf("CryptProtectData failed: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(output.pbData)))

	encrypted := make([]byte, output.cbData)
	copy(encrypted, output.toByteSlice())
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// Decrypt decrypts a base64-encoded DPAPI-encrypted string back to plain-text.
func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		// If it cannot be decoded as base64, it might be a plain-text legacy value.
		// Return as-is so migration can handle it gracefully.
		return ciphertext, nil
	}

	input := newBlob(data)
	var output dataBlob

	r, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(input)),
		0, // description
		0, // optional entropy
		0, // reserved
		0, // prompt struct
		0, // flags
		uintptr(unsafe.Pointer(&output)),
	)
	if r == 0 {
		// Decryption failed — value is likely plain-text (legacy). Return as-is.
		return ciphertext, nil
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(output.pbData)))

	decrypted := make([]byte, output.cbData)
	copy(decrypted, output.toByteSlice())
	return string(decrypted), nil
}

// IsEncrypted checks if a string looks like a DPAPI-encrypted base64 blob.
// Used during migration to avoid double-encrypting.
func IsEncrypted(value string) bool {
	if value == "" {
		return false
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(data) < 20 {
		return false
	}
	// Try to decrypt — if it succeeds and returns different content, it's encrypted.
	decrypted, err := Decrypt(value)
	if err != nil {
		return false
	}
	return decrypted != value
}
