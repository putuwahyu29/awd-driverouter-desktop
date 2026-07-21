package provider

import (
	"io"
	"time"
)

// FileMetadata represents standardized metadata for a file or folder across all cloud providers.
type FileMetadata struct {
	ID          string    `json:"id"`           // Virtual UUID
	Name        string    `json:"name"`         // Name of the file/folder
	Size        int64     `json:"size"`         // Size in bytes
	IsFolder    bool      `json:"isFolder"`     // True if directory
	ParentID    string    `json:"parentId"`     // Parent Virtual UUID
	Provider    string    `json:"provider"`     // provider type: google, onedrive, dropbox
	AccountID   string    `json:"accountId"`    // The unique database ID of the linked account
	PhysicalID  string    `json:"physicalId"`   // The ID assigned by the actual cloud provider
	CreatedAt   time.Time `json:"createdAt"`
	ModifiedAt  time.Time `json:"modifiedAt"`
	Starred     bool      `json:"starred"`
	Shared      bool      `json:"shared"`
}

// AccountInfo represents metadata about a linked cloud storage account.
type AccountInfo struct {
	ID           string `json:"id"`
	Provider     string `json:"provider"` // google, onedrive, dropbox
	DisplayName  string `json:"displayName"`
	Email        string `json:"email"`
	UsedSpace    int64  `json:"usedSpace"`
	TotalSpace   int64  `json:"totalSpace"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	TokenExpiry  string `json:"tokenExpiry"`
	Active       bool   `json:"active"`
}

// Provider defines the interface that all cloud storage adapters must implement.
type Provider interface {
	GetUserInfo() (name, email string, err error)
	GetQuota() (used, total int64, err error)
	ListDirectory(remoteParentID string) ([]FileMetadata, error)
	UploadFile(remoteParentID string, filename string, r io.Reader, size int64) (physicalID string, err error)
	DownloadFile(physicalID string) (io.ReadCloser, error)
	DeleteFile(physicalID string) error
	CreateFolder(remoteParentID string, name string) (physicalID string, err error)
}

type SharePermission struct {
	ID           string `json:"id"`
	Type         string `json:"type"` // "user", "anyone", "group"
	Role         string `json:"role"` // "reader" (Viewer), "writer" (Editor), "owner"
	EmailAddress string `json:"emailAddress,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
}

type ShareManager interface {
	ListPermissions(physicalID string) ([]SharePermission, error)
	AddPermission(physicalID string, email string, role string) error
	DeletePermission(physicalID string, permID string) error
	SetGeneralAccess(physicalID string, accessType string, role string) error
}
