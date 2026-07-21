package provider

import (
	"fmt"
	"io"
	"net"
	"path"
	"strings"
	"time"

	"github.com/hirochachacha/go-smb2"
)

// SMBProvider implements Provider interface for Windows Network Shared Folders (LAN SMB/CIFS).
type SMBProvider struct {
	Host     string
	Share    string
	Username string
	Password string
	Domain   string
}

func NewSMBProvider(host, share, username, password string) *SMBProvider {
	domain := ""
	if strings.Contains(username, "\\") {
		parts := strings.SplitN(username, "\\", 2)
		domain = parts[0]
		username = parts[1]
	}
	return &SMBProvider{
		Host:     host,
		Share:    share,
		Username: username,
		Password: password,
		Domain:   domain,
	}
}

func (s *SMBProvider) connect() (net.Conn, *smb2.Session, *smb2.Share, error) {
	port := 445
	if !strings.Contains(s.Host, ":") {
		s.Host = fmt.Sprintf("%s:%d", s.Host, port)
	}

	conn, err := net.DialTimeout("tcp", s.Host, 10*time.Second)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("SMB dial failed: %w", err)
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     s.Username,
			Password: s.Password,
			Domain:   s.Domain,
		},
	}

	sess, err := d.Dial(conn)
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("SMB NTLM authentication failed: %w", err)
	}

	share, err := sess.Mount(s.Share)
	if err != nil {
		sess.Logoff()
		conn.Close()
		return nil, nil, nil, fmt.Errorf("SMB mount share '\\\\%s\\%s' failed: %w", s.Host, s.Share, err)
	}

	return conn, sess, share, nil
}

func (s *SMBProvider) GetUserInfo() (name, email string, err error) {
	return fmt.Sprintf("SMB Share (\\\\%s\\%s)", s.Host, s.Share), s.Username, nil
}

func (s *SMBProvider) GetQuota() (used, total int64, err error) {
	conn, sess, share, err := s.connect()
	if err != nil {
		return 0, 500 * 1024 * 1024 * 1024, nil
	}
	defer share.Umount()
	defer sess.Logoff()
	defer conn.Close()

	// Return 500 GB default storage display for LAN SMB Shares
	return 0, 500 * 1024 * 1024 * 1024, nil

	return 0, 500 * 1024 * 1024 * 1024, nil
}

func (s *SMBProvider) ListDirectory(physicalFolderID string) ([]FileMetadata, error) {
	conn, sess, share, err := s.connect()
	if err != nil {
		return nil, err
	}
	defer share.Umount()
	defer sess.Logoff()
	defer conn.Close()

	targetDir := physicalFolderID
	if targetDir == "" || targetDir == "root" {
		targetDir = "."
	}
	targetDir = strings.ReplaceAll(targetDir, "/", "\\")

	entries, err := share.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("SMB read dir failed: %w", err)
	}

	var results []FileMetadata
	for _, entry := range entries {
		if entry.Name() == "." || entry.Name() == ".." {
			continue
		}

		itemPath := path.Join(physicalFolderID, entry.Name())
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

func (s *SMBProvider) CreateFolder(physicalParentID string, name string) (string, error) {
	conn, sess, share, err := s.connect()
	if err != nil {
		return "", err
	}
	defer share.Umount()
	defer sess.Logoff()
	defer conn.Close()

	parent := physicalParentID
	if parent == "" || parent == "root" {
		parent = ""
	}
	folderPath := path.Join(parent, name)
	winPath := strings.ReplaceAll(folderPath, "/", "\\")

	err = share.Mkdir(winPath, 0755)
	if err != nil {
		return "", fmt.Errorf("SMB mkdir failed: %w", err)
	}

	return folderPath, nil
}

func (s *SMBProvider) UploadFile(physicalParentID string, filename string, r io.Reader, size int64) (string, error) {
	conn, sess, share, err := s.connect()
	if err != nil {
		return "", err
	}
	defer share.Umount()
	defer sess.Logoff()
	defer conn.Close()

	parent := physicalParentID
	if parent == "" || parent == "root" {
		parent = ""
	}
	filePath := path.Join(parent, filename)
	winPath := strings.ReplaceAll(filePath, "/", "\\")

	f, err := share.Create(winPath)
	if err != nil {
		return "", fmt.Errorf("SMB create file failed: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	if err != nil {
		return "", fmt.Errorf("SMB write file failed: %w", err)
	}

	return filePath, nil
}

func (s *SMBProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	conn, sess, share, err := s.connect()
	if err != nil {
		return nil, err
	}

	winPath := strings.ReplaceAll(physicalID, "/", "\\")
	f, err := share.Open(winPath)
	if err != nil {
		share.Umount()
		sess.Logoff()
		conn.Close()
		return nil, fmt.Errorf("SMB open file failed: %w", err)
	}

	return &smbReadCloser{File: f, share: share, sess: sess, conn: conn}, nil
}

type smbReadCloser struct {
	*smb2.File
	share *smb2.Share
	sess  *smb2.Session
	conn  net.Conn
}

func (s *smbReadCloser) Close() error {
	err := s.File.Close()
	s.share.Umount()
	s.sess.Logoff()
	s.conn.Close()
	return err
}

func (s *SMBProvider) DeleteFile(physicalID string) error {
	conn, sess, share, err := s.connect()
	if err != nil {
		return err
	}
	defer share.Umount()
	defer sess.Logoff()
	defer conn.Close()

	winPath := strings.ReplaceAll(physicalID, "/", "\\")
	stat, err := share.Stat(winPath)
	if err == nil && stat.IsDir() {
		return share.RemoveAll(winPath)
	}
	return share.Remove(winPath)
}

func (s *SMBProvider) RenameFile(physicalID string, newName string) error {
	conn, sess, share, err := s.connect()
	if err != nil {
		return err
	}
	defer share.Umount()
	defer sess.Logoff()
	defer conn.Close()

	oldWinPath := strings.ReplaceAll(physicalID, "/", "\\")
	parentDir := path.Dir(physicalID)
	newPath := path.Join(parentDir, newName)
	newWinPath := strings.ReplaceAll(newPath, "/", "\\")

	return share.Rename(oldWinPath, newWinPath)
}

func (s *SMBProvider) GetOAuthURL() string {
	return ""
}
