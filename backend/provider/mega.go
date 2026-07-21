package provider

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/t3rm1n4l/go-mega"
)

// MegaProvider implements the Provider interface for MEGA storage.
type MegaProvider struct {
	client *mega.Mega
	email  string
}

// NewMegaProvider initializes a new MEGA client session using email & password
func NewMegaProvider(email, password string) (*MegaProvider, error) {
	m := mega.New()
	err := m.Login(email, password)
	if err != nil {
		return nil, fmt.Errorf("failed to login to MEGA: %w", err)
	}
	return &MegaProvider{
		client: m,
		email:  email,
	}, nil
}

func (m *MegaProvider) GetUserInfo() (name, email string, err error) {
	user, err := m.client.GetUser()
	if err != nil {
		return "MEGA User", m.email, nil
	}
	name = user.Name
	if name == "" {
		name = m.email
	}
	return name, m.email, nil
}

func (m *MegaProvider) GetQuota() (usedSpace, totalSpace int64, err error) {
	quota, err := m.client.GetQuota()
	if err != nil {
		return 0, 0, err
	}
	return int64(quota.Cstrg), int64(quota.Mstrg), nil
}

func (m *MegaProvider) ListDirectory(physicalFolderID string) ([]FileMetadata, error) {
	var targetNode *mega.Node
	if physicalFolderID == "" || physicalFolderID == "0" || physicalFolderID == "root" {
		targetNode = m.client.FS.GetRoot()
	} else {
		nodes, err := m.client.FS.GetChildren(m.client.FS.GetRoot())
		if err != nil {
			return nil, err
		}
		for _, n := range nodes {
			if n.GetHash() == physicalFolderID {
				targetNode = n
				break
			}
		}
	}

	if targetNode == nil {
		targetNode = m.client.FS.GetRoot()
	}

	children, err := m.client.FS.GetChildren(targetNode)
	if err != nil {
		return nil, err
	}

	var results []FileMetadata
	for _, node := range children {
		isDir := node.GetType() == mega.FOLDER
		mTime := node.GetTimeStamp()

		results = append(results, FileMetadata{
			PhysicalID: node.GetHash(),
			Name:       node.GetName(),
			Size:       node.GetSize(),
			IsFolder:   isDir,
			ModifiedAt: mTime,
		})
	}

	return results, nil
}

func (m *MegaProvider) CreateFolder(physicalParentID string, name string) (string, error) {
	parent := m.client.FS.GetRoot()
	if physicalParentID != "" && physicalParentID != "root" && physicalParentID != "0" {
		nodes, err := m.client.FS.GetChildren(parent)
		if err == nil {
			for _, n := range nodes {
				if n.GetHash() == physicalParentID {
					parent = n
					break
				}
			}
		}
	}

	node, err := m.client.CreateDir(name, parent)
	if err != nil {
		return "", err
	}
	return node.GetHash(), nil
}

func (m *MegaProvider) UploadFile(physicalParentID string, filename string, r io.Reader, size int64) (string, error) {
	parent := m.client.FS.GetRoot()
	if physicalParentID != "" && physicalParentID != "root" && physicalParentID != "0" {
		nodes, err := m.client.FS.GetChildren(parent)
		if err == nil {
			for _, n := range nodes {
				if n.GetHash() == physicalParentID {
					parent = n
					break
				}
			}
		}
	}

	tmpFile, err := os.CreateTemp("", "mega_upload_*_"+filename)
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, r); err != nil {
		tmpFile.Close()
		return "", err
	}
	tmpFile.Close()

	node, err := m.client.UploadFile(tmpPath, parent, filename, nil)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to MEGA: %w", err)
	}

	return node.GetHash(), nil
}

func (m *MegaProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	nodes, err := m.client.FS.GetChildren(m.client.FS.GetRoot())
	if err != nil {
		return nil, err
	}
	var target *mega.Node
	for _, n := range nodes {
		if n.GetHash() == physicalID {
			target = n
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("file not found on MEGA: %s", physicalID)
	}

	tmpFile, err := os.CreateTemp("", "mega_download_*_"+filepath.Base(target.GetName()))
	if err != nil {
		return nil, err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	err = m.client.DownloadFile(target, tmpPath, nil)
	if err != nil {
		os.Remove(tmpPath)
		return nil, err
	}

	f, err := os.Open(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return nil, err
	}

	return &autoDeleteFileReader{File: f, filePath: tmpPath}, nil
}

type autoDeleteFileReader struct {
	*os.File
	filePath string
}

func (a *autoDeleteFileReader) Close() error {
	err := a.File.Close()
	os.Remove(a.filePath)
	return err
}

func (m *MegaProvider) DeleteFile(physicalID string) error {
	nodes, err := m.client.FS.GetChildren(m.client.FS.GetRoot())
	if err != nil {
		return err
	}
	var target *mega.Node
	for _, n := range nodes {
		if n.GetHash() == physicalID {
			target = n
			break
		}
	}
	if target == nil {
		return fmt.Errorf("file not found on MEGA: %s", physicalID)
	}

	return m.client.Delete(target, true)
}

func (m *MegaProvider) RenameFile(physicalID string, newName string) error {
	nodes, err := m.client.FS.GetChildren(m.client.FS.GetRoot())
	if err != nil {
		return err
	}
	var target *mega.Node
	for _, n := range nodes {
		if n.GetHash() == physicalID {
			target = n
			break
		}
	}
	if target == nil {
		return fmt.Errorf("file not found on MEGA: %s", physicalID)
	}

	return m.client.Rename(target, newName)
}

func (m *MegaProvider) GetOAuthURL() string {
	return ""
}
