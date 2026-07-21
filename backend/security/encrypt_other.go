//go:build !windows

package security

// Encrypt is a no-op on non-Windows platforms.
// On macOS and Linux, consider using the system keychain instead.
// For now, returns the plaintext unchanged (no DPAPI available).
func Encrypt(plaintext string) (string, error) {
	return plaintext, nil
}

// Decrypt is a no-op on non-Windows platforms.
func Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

// IsEncrypted always returns false on non-Windows platforms.
func IsEncrypted(value string) bool {
	return false
}
