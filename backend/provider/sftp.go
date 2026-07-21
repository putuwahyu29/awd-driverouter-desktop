package provider

import (
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTPProvider implements the Provider interface for SFTP (SSH) servers.
type SFTPProvider struct {
	Host     string
	Port     int
	Username string
	Password string
	BaseDir  string
}

func NewSFTPProvider(host string, port int, username, password, baseDir string) *SFTPProvider {
	if port <= 0 {
		port = 22
	}
	if baseDir == "" {
		baseDir = "/"
	}
	return &SFTPProvider{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		BaseDir:  baseDir,
	}
}

func (s *SFTPProvider) connect() (*ssh.Client, *sftp.Client, error) {
	config := &ssh.ClientConfig{
		User: s.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	sshConn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, nil, fmt.Errorf("SFTP SSH connection failed: %w", err)
	}

	client, err := sftp.NewClient(sshConn)
	if err != nil {
		sshConn.Close()
		return nil, nil, fmt.Errorf("SFTP client initialization failed: %w", err)
	}

	return sshConn, client, nil
}

func (s *SFTPProvider) GetUserInfo() (name, email string, err error) {
	return fmt.Sprintf("SFTP (%s)", s.Host), s.Username, nil
}

func (s *SFTPProvider) GetQuota() (used, total int64, err error) {
	sshConn, client, err := s.connect()
	if err != nil {
		return 0, 100 * 1024 * 1024 * 1024, nil
	}
	defer sshConn.Close()
	defer client.Close()

	stat, err := client.StatVFS(s.BaseDir)
	if err == nil && stat != nil {
		totalBytes := int64(stat.Blocks) * int64(stat.Bsize)
		freeBytes := int64(stat.Bfree) * int64(stat.Bsize)
		usedBytes := totalBytes - freeBytes
		if totalBytes > 0 {
			return usedBytes, totalBytes, nil
		}
	}

	return 0, 100 * 1024 * 1024 * 1024, nil
}

func (s *SFTPProvider) ListDirectory(physicalFolderID string) ([]FileMetadata, error) {
	sshConn, client, err := s.connect()
	if err != nil {
		return nil, err
	}
	defer sshConn.Close()
	defer client.Close()

	targetDir := physicalFolderID
	if targetDir == "" || targetDir == "root" {
		targetDir = s.BaseDir
	}
	if !strings.HasPrefix(targetDir, "/") {
		targetDir = "/" + targetDir
	}

	entries, err := client.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("SFTP read dir failed: %w", err)
	}

	var results []FileMetadata
	for _, entry := range entries {
		if entry.Name() == "." || entry.Name() == ".." {
			continue
		}

		itemPath := path.Join(targetDir, entry.Name())
		results = append(results, FileMetadata{
			PhysicalID: itemPath,
			Name:       entry.Name(),
			Size:       entry.Size(),
			IsFolder:   entry.IsDir(),
			ModifiedAt: entry.ModTime(),
		})
	}

	return results, nil
}

func (s *SFTPProvider) CreateFolder(physicalParentID string, name string) (string, error) {
	sshConn, client, err := s.connect()
	if err != nil {
		return "", err
	}
	defer sshConn.Close()
	defer client.Close()

	parent := physicalParentID
	if parent == "" || parent == "root" {
		parent = s.BaseDir
	}

	folderPath := path.Join(parent, name)
	err = client.MkdirAll(folderPath)
	if err != nil {
		return "", fmt.Errorf("SFTP mkdir failed: %w", err)
	}

	return folderPath, nil
}

func (s *SFTPProvider) UploadFile(physicalParentID string, filename string, r io.Reader, size int64) (string, error) {
	sshConn, client, err := s.connect()
	if err != nil {
		return "", err
	}
	defer sshConn.Close()
	defer client.Close()

	parent := physicalParentID
	if parent == "" || parent == "root" {
		parent = s.BaseDir
	}

	filePath := path.Join(parent, filename)
	f, err := client.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("SFTP file create failed: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	if err != nil {
		return "", fmt.Errorf("SFTP file write failed: %w", err)
	}

	return filePath, nil
}

func (s *SFTPProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	sshConn, client, err := s.connect()
	if err != nil {
		return nil, err
	}

	f, err := client.Open(physicalID)
	if err != nil {
		client.Close()
		sshConn.Close()
		return nil, fmt.Errorf("SFTP open file failed: %w", err)
	}

	return &sftpReadCloser{File: f, client: client, sshConn: sshConn}, nil
}

type sftpReadCloser struct {
	*sftp.File
	client  *sftp.Client
	sshConn *ssh.Client
}

func (s *sftpReadCloser) Close() error {
	err := s.File.Close()
	s.client.Close()
	s.sshConn.Close()
	return err
}

func (s *SFTPProvider) DeleteFile(physicalID string) error {
	sshConn, client, err := s.connect()
	if err != nil {
		return err
	}
	defer sshConn.Close()
	defer client.Close()

	stat, err := client.Stat(physicalID)
	if err == nil && stat.IsDir() {
		return client.RemoveAll(physicalID)
	}
	return client.Remove(physicalID)
}

func (s *SFTPProvider) RenameFile(physicalID string, newName string) error {
	sshConn, client, err := s.connect()
	if err != nil {
		return err
	}
	defer sshConn.Close()
	defer client.Close()

	parentDir := path.Dir(physicalID)
	newPath := path.Join(parentDir, newName)
	return client.Rename(physicalID, newPath)
}

func (s *SFTPProvider) GetOAuthURL() string {
	return ""
}
