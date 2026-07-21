//go:build windows

package security

import (
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	cases := []struct {
		name      string
		plaintext string
	}{
		{"normal token", "ya29.a0AfB_byC-someOAuthAccessToken_123456"},
		{"empty string", ""},
		{"short string", "abc"},
		{"password with special chars", "P@ssw0rd!#$%^&*()"},
		{"long token", "eyJhbGciOiJSUzI1NiIsImtpZCI6IjEifQ.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := Encrypt(tc.plaintext)
			if err != nil {
				t.Fatalf("Encrypt(%q) returned error: %v", tc.plaintext, err)
			}

			decrypted, err := Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt returned error: %v", err)
			}

			if decrypted != tc.plaintext {
				t.Errorf("Roundtrip failed: got %q, want %q", decrypted, tc.plaintext)
			}
		})
	}
}

func TestEncryptProducesDifferentOutput(t *testing.T) {
	plaintext := "test-secret-token"
	encrypted, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if plaintext == "" {
		return
	}

	if encrypted == plaintext {
		t.Error("Encrypt should produce different output than plaintext")
	}
}

func TestDecryptPlaintextFallback(t *testing.T) {
	// Legacy plain-text tokens should be returned as-is (migration safety)
	legacy := "plain-text-token-not-encrypted"
	result, err := Decrypt(legacy)
	if err != nil {
		t.Fatalf("Decrypt of plain-text returned error: %v", err)
	}
	if result != legacy {
		t.Errorf("Expected plain-text fallback, got %q", result)
	}
}

func TestIsEncrypted(t *testing.T) {
	plain := "not-encrypted-at-all"
	if IsEncrypted(plain) {
		t.Error("Expected IsEncrypted to return false for plain text")
	}

	encrypted, _ := Encrypt("some-token")
	if encrypted != "" && !IsEncrypted(encrypted) {
		t.Error("Expected IsEncrypted to return true for DPAPI-encrypted value")
	}
}
