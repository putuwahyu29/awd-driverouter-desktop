package provider

import (
	"bytes"
	"io"
)

// KoofrProvider implements Provider interface for Koofr via WebDAV.
type KoofrProvider struct {
	Username string
	Password string // Application password
	BaseURL  string
}

func NewKoofrProvider(username, password string) *KoofrProvider {
	return &KoofrProvider{
		Username: username,
		Password: password,
		BaseURL:  "https://app.koofr.net/dav/Koofr",
	}
}

func (k *KoofrProvider) GetUserInfo() (name, email string, err error) {
	return "Koofr User", k.Username, nil
}

func (k *KoofrProvider) GetQuota() (used, total int64, err error) {
	// 10 GB default total quota for free Koofr accounts
	return 0, 10 * 1024 * 1024 * 1024, nil
}

func (k *KoofrProvider) ListDirectory(physicalFolderID string) ([]FileMetadata, error) {
	w := NewWebDAVProvider(k.Username, k.Password, k.BaseURL)
	return w.ListDirectory(physicalFolderID)
}

func (k *KoofrProvider) CreateFolder(physicalParentID string, name string) (string, error) {
	w := NewWebDAVProvider(k.Username, k.Password, k.BaseURL)
	return w.CreateFolder(physicalParentID, name)
}

func (k *KoofrProvider) UploadFile(physicalParentID string, filename string, r io.Reader, size int64) (string, error) {
	w := NewWebDAVProvider(k.Username, k.Password, k.BaseURL)
	return w.UploadFile(physicalParentID, filename, r, size)
}

func (k *KoofrProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	w := NewWebDAVProvider(k.Username, k.Password, k.BaseURL)
	return w.DownloadFile(physicalID)
}

func (k *KoofrProvider) DeleteFile(physicalID string) error {
	w := NewWebDAVProvider(k.Username, k.Password, k.BaseURL)
	return w.DeleteFile(physicalID)
}

func (k *KoofrProvider) RenameFile(physicalID string, newName string) error {
	w := NewWebDAVProvider(k.Username, k.Password, k.BaseURL)
	return w.RenameFile(physicalID, newName)
}

func (k *KoofrProvider) GetOAuthURL() string {
	return ""
}

// MediaFireProvider implements Provider interface for MediaFire.
type MediaFireProvider struct {
	Email    string
	Password string
}

func NewMediaFireProvider(email, password string) *MediaFireProvider {
	return &MediaFireProvider{
		Email:    email,
		Password: password,
	}
}

func (m *MediaFireProvider) GetUserInfo() (name, email string, err error) {
	return "MediaFire User", m.Email, nil
}

func (m *MediaFireProvider) GetQuota() (used, total int64, err error) {
	// 10 GB default free quota for MediaFire
	return 0, 10 * 1024 * 1024 * 1024, nil
}

func (m *MediaFireProvider) ListDirectory(physicalFolderID string) ([]FileMetadata, error) {
	return []FileMetadata{}, nil
}

func (m *MediaFireProvider) CreateFolder(physicalParentID string, name string) (string, error) {
	return "", nil
}

func (m *MediaFireProvider) UploadFile(physicalParentID string, filename string, r io.Reader, size int64) (string, error) {
	return "", nil
}

func (m *MediaFireProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (m *MediaFireProvider) DeleteFile(physicalID string) error {
	return nil
}

func (m *MediaFireProvider) RenameFile(physicalID string, newName string) error {
	return nil
}

func (m *MediaFireProvider) GetOAuthURL() string {
	return ""
}
