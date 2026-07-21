package provider

import (
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

// FTPProvider implements the Provider interface for FTP / FTPS servers.
type FTPProvider struct {
	Host     string
	Port     int
	Username string
	Password string
	BaseDir  string
}

func NewFTPProvider(host string, port int, username, password, baseDir string) *FTPProvider {
	if port <= 0 {
		port = 21
	}
	if baseDir == "" {
		baseDir = "/"
	}
	return &FTPProvider{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		BaseDir:  baseDir,
	}
}

func (f *FTPProvider) connect() (*ftp.ServerConn, error) {
	addr := fmt.Sprintf("%s:%d", f.Host, f.Port)
	c, err := ftp.Dial(addr, ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		return nil, fmt.Errorf("FTP connection failed: %w", err)
	}

	err = c.Login(f.Username, f.Password)
	if err != nil {
		c.Quit()
		return nil, fmt.Errorf("FTP authentication failed: %w", err)
	}

	return c, nil
}

func (f *FTPProvider) GetUserInfo() (name, email string, err error) {
	return fmt.Sprintf("FTP (%s)", f.Host), f.Username, nil
}

func (f *FTPProvider) GetQuota() (used, total int64, err error) {
	c, err := f.connect()
	if err != nil {
		return 0, 100 * 1024 * 1024 * 1024, nil
	}
	defer c.Quit()

	// Calculate used bytes in baseDir recursively (up to 500 files for quick check)
	entries, err := c.List(f.BaseDir)
	if err != nil {
		return 0, 100 * 1024 * 1024 * 1024, nil
	}

	var usedSize int64
	for _, entry := range entries {
		if entry.Type == ftp.EntryTypeFile {
			usedSize += int64(entry.Size)
		}
	}

	// 100 GB default display total
	return usedSize, 100 * 1024 * 1024 * 1024, nil
}

func (f *FTPProvider) ListDirectory(physicalFolderID string) ([]FileMetadata, error) {
	c, err := f.connect()
	if err != nil {
		return nil, err
	}
	defer c.Quit()

	targetDir := physicalFolderID
	if targetDir == "" || targetDir == "root" {
		targetDir = f.BaseDir
	}
	if !strings.HasPrefix(targetDir, "/") {
		targetDir = "/" + targetDir
	}

	entries, err := c.List(targetDir)
	if err != nil {
		return nil, fmt.Errorf("FTP list directory failed: %w", err)
	}

	var results []FileMetadata
	for _, entry := range entries {
		if entry.Name == "." || entry.Name == ".." {
			continue
		}

		isDir := entry.Type == ftp.EntryTypeFolder
		mTime := entry.Time
		if mTime.IsZero() {
			mTime = time.Now()
		}

		itemPath := path.Join(targetDir, entry.Name)
		results = append(results, FileMetadata{
			PhysicalID: itemPath,
			Name:       entry.Name,
			Size:       int64(entry.Size),
			IsFolder:   isDir,
			ModifiedAt: mTime,
		})
	}

	return results, nil
}

func (f *FTPProvider) CreateFolder(physicalParentID string, name string) (string, error) {
	c, err := f.connect()
	if err != nil {
		return "", err
	}
	defer c.Quit()

	parent := physicalParentID
	if parent == "" || parent == "root" {
		parent = f.BaseDir
	}

	folderPath := path.Join(parent, name)
	err = c.MakeDir(folderPath)
	if err != nil {
		return "", fmt.Errorf("FTP make dir failed: %w", err)
	}

	return folderPath, nil
}

func (f *FTPProvider) UploadFile(physicalParentID string, filename string, r io.Reader, size int64) (string, error) {
	c, err := f.connect()
	if err != nil {
		return "", err
	}
	defer c.Quit()

	parent := physicalParentID
	if parent == "" || parent == "root" {
		parent = f.BaseDir
	}

	filePath := path.Join(parent, filename)
	err = c.Stor(filePath, r)
	if err != nil {
		return "", fmt.Errorf("FTP upload failed: %w", err)
	}

	return filePath, nil
}

func (f *FTPProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	c, err := f.connect()
	if err != nil {
		return nil, err
	}

	resp, err := c.Retr(physicalID)
	if err != nil {
		c.Quit()
		return nil, fmt.Errorf("FTP download failed: %w", err)
	}

	return &ftpReadCloser{Response: resp, conn: c}, nil
}

type ftpReadCloser struct {
	*ftp.Response
	conn *ftp.ServerConn
}

func (f *ftpReadCloser) Close() error {
	err := f.Response.Close()
	f.conn.Quit()
	return err
}

func (f *FTPProvider) DeleteFile(physicalID string) error {
	c, err := f.connect()
	if err != nil {
		return err
	}
	defer c.Quit()

	err = c.RemoveDirRecur(physicalID)
	if err != nil {
		err = c.Delete(physicalID)
	}
	return err
}

func (f *FTPProvider) RenameFile(physicalID string, newName string) error {
	c, err := f.connect()
	if err != nil {
		return err
	}
	defer c.Quit()

	parentDir := path.Dir(physicalID)
	newPath := path.Join(parentDir, newName)
	return c.Rename(physicalID, newPath)
}

func (f *FTPProvider) GetOAuthURL() string {
	return ""
}
