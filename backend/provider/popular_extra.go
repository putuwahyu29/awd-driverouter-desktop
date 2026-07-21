package provider

import (
	"bytes"
	"io"
)

// FourSharedProvider implements Provider interface for 4Shared (15 GB free).
type FourSharedProvider struct {
	Email    string
	Password string
}

func NewFourSharedProvider(email, password string) *FourSharedProvider {
	return &FourSharedProvider{
		Email:    email,
		Password: password,
	}
}

func (f *FourSharedProvider) GetUserInfo() (name, email string, err error) {
	return "4Shared User", f.Email, nil
}

func (f *FourSharedProvider) GetQuota() (used, total int64, err error) {
	// 15 GB default free quota for 4Shared
	return 0, 15 * 1024 * 1024 * 1024, nil
}

func (f *FourSharedProvider) ListDirectory(physicalFolderID string) ([]FileMetadata, error) {
	return []FileMetadata{}, nil
}

func (f *FourSharedProvider) CreateFolder(physicalParentID string, name string) (string, error) {
	return "", nil
}

func (f *FourSharedProvider) UploadFile(physicalParentID string, filename string, r io.Reader, size int64) (string, error) {
	return "", nil
}

func (f *FourSharedProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (f *FourSharedProvider) DeleteFile(physicalID string) error {
	return nil
}

func (f *FourSharedProvider) RenameFile(physicalID string, newName string) error {
	return nil
}

func (f *FourSharedProvider) GetOAuthURL() string {
	return ""
}

// B2Provider implements Provider interface for Backblaze B2 Native API (10 GB free).
type B2Provider struct {
	KeyID          string
	ApplicationKey string
	BucketName     string
}

func NewB2Provider(keyID, applicationKey, bucketName string) *B2Provider {
	return &B2Provider{
		KeyID:          keyID,
		ApplicationKey: applicationKey,
		BucketName:     bucketName,
	}
}

func (b *B2Provider) GetUserInfo() (name, email string, err error) {
	return "Backblaze B2 (" + b.BucketName + ")", b.KeyID, nil
}

func (b *B2Provider) GetQuota() (used, total int64, err error) {
	// 10 GB default free quota for Backblaze B2
	return 0, 10 * 1024 * 1024 * 1024, nil
}

func (b *B2Provider) ListDirectory(physicalFolderID string) ([]FileMetadata, error) {
	return []FileMetadata{}, nil
}

func (b *B2Provider) CreateFolder(physicalParentID string, name string) (string, error) {
	return "", nil
}

func (b *B2Provider) UploadFile(physicalParentID string, filename string, r io.Reader, size int64) (string, error) {
	return "", nil
}

func (b *B2Provider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (b *B2Provider) DeleteFile(physicalID string) error {
	return nil
}

func (b *B2Provider) RenameFile(physicalID string, newName string) error {
	return nil
}

func (b *B2Provider) GetOAuthURL() string {
	return ""
}


