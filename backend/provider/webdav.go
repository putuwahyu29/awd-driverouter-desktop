package provider

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

// WebDAVProvider implements the Provider interface for WebDAV storage.
// It maps credentials as follows:
// - Email: Username
// - AccessToken: Password / App Token
// - RefreshToken: Base Server URL (e.g., https://example.com/remote.php/dav/files/user/)
type WebDAVProvider struct {
	Username string
	Password string
	BaseURL  string
}

// NewWebDAVProvider creates a new instance of WebDAVProvider
func NewWebDAVProvider(username, password, baseURL string) *WebDAVProvider {
	// Clean up baseURL to ensure it has no trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &WebDAVProvider{
		Username: username,
		Password: password,
		BaseURL:  baseURL,
	}
}

// XML structures for PROPFIND response parsing
type WebDAVMultiStatus struct {
	XMLName   xml.Name         `xml:"multistatus"`
	Responses []WebDAVResponse `xml:"response"`
}

type WebDAVResponse struct {
	Href     string         `xml:"href"`
	Propstat WebDAVPropstat `xml:"propstat"`
}

type WebDAVPropstat struct {
	Prop WebDAVProp `xml:"prop"`
}

type WebDAVProp struct {
	ResourceType WebDAVResourceType `xml:"resourcetype"`
	ContentLength string            `xml:"getcontentlength"`
	LastModified  string            `xml:"getlastmodified"`
	DisplayName   string            `xml:"displayname"`
}

type WebDAVResourceType struct {
	Collection *struct{} `xml:"collection"`
}

func (w *WebDAVProvider) doRequest(method, relativePath string, body io.Reader, headers map[string]string) (*http.Response, error) {
	// Ensure relativePath starts with a slash
	if !strings.HasPrefix(relativePath, "/") {
		relativePath = "/" + relativePath
	}
	
	// Escape path segments properly
	u, err := url.Parse(w.BaseURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, relativePath)

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(w.Username, w.Password)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (w *WebDAVProvider) GetQuota() (usedSpace, totalSpace int64, err error) {
	// Send PROPFIND request on root path to extract quota properties
	body := `<?xml version="1.0" encoding="utf-8" ?>
<d:propfind xmlns:d="DAV:">
  <d:prop>
    <d:quota-available-bytes/>
    <d:quota-used-bytes/>
  </d:prop>
</d:propfind>`

	headers := map[string]string{
		"Content-Type": "application/xml; charset=utf-8",
		"Depth":        "0",
	}

	resp, err := w.doRequest("PROPFIND", "/", bytes.NewBufferString(body), headers)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("webdav quota failed with status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}

	// Simple parse of quota tags
	used := int64(0)
	available := int64(0)
	
	usedMatch := strings.Contains(string(respBody), "quota-used-bytes")
	if usedMatch {
		type QuotaProp struct {
			Used      int64 `xml:"quota-used-bytes"`
			Available int64 `xml:"quota-available-bytes"`
		}
		type QuotaPropstat struct {
			Prop QuotaProp `xml:"prop"`
		}
		type QuotaResponse struct {
			Propstat QuotaPropstat `xml:"propstat"`
		}
		type QuotaMultiStatus struct {
			XMLName   xml.Name        `xml:"multistatus"`
			Responses []QuotaResponse `xml:"response"`
		}
		var qms QuotaMultiStatus
		if err := xml.Unmarshal(respBody, &qms); err == nil && len(qms.Responses) > 0 {
			used = qms.Responses[0].Propstat.Prop.Used
			available = qms.Responses[0].Propstat.Prop.Available
		}
	}

	// Total quota is used + available
	return used, used + available, nil
}

func (w *WebDAVProvider) GetUserInfo() (name, email string, err error) {
	return "WebDAV User", w.Username, nil
}

func (w *WebDAVProvider) ListDirectory(physicalFolderID string) ([]FileMetadata, error) {
	// physicalFolderID is the relative WebDAV path
	pathLoc := physicalFolderID
	if pathLoc == "root" || pathLoc == "" {
		pathLoc = "/"
	}

	headers := map[string]string{
		"Depth": "1",
	}

	resp, err := w.doRequest("PROPFIND", pathLoc, nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("webdav list failed with status %d", resp.StatusCode)
	}

	var ms WebDAVMultiStatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, fmt.Errorf("failed to parse webdav xml response: %w", err)
	}

	var items []FileMetadata
	
	// Prepare base url parser to clean href paths
	baseParsed, _ := url.Parse(w.BaseURL)
	basePrefix := baseParsed.Path

	for _, r := range ms.Responses {
		// Clean href from URL escapes
		decodedHref, err := url.PathUnescape(r.Href)
		if err != nil {
			decodedHref = r.Href
		}

		// Ensure it represents a relative subpath of base URL
		rel := strings.TrimPrefix(decodedHref, basePrefix)
		rel = "/" + strings.TrimPrefix(rel, "/")

		// Propfinddepth=1 returns the parent folder itself as the first response element. Skip it.
		parentRel := strings.TrimSuffix(pathLoc, "/")
		cleanedRel := strings.TrimSuffix(rel, "/")
		if parentRel == cleanedRel {
			continue
		}

		name := r.Propstat.Prop.DisplayName
		if name == "" {
			name = path.Base(rel)
		}

		isFolder := r.Propstat.Prop.ResourceType.Collection != nil
		size, _ := strconv.ParseInt(r.Propstat.Prop.ContentLength, 10, 64)
		
		modifiedTime := time.Now()
		if r.Propstat.Prop.LastModified != "" {
			if t, err := time.Parse(time.RFC1123, r.Propstat.Prop.LastModified); err == nil {
				modifiedTime = t
			} else if t, err := time.Parse(time.RFC3339, r.Propstat.Prop.LastModified); err == nil {
				modifiedTime = t
			}
		}

		items = append(items, FileMetadata{
			PhysicalID: rel,
			Name:       name,
			Size:       size,
			IsFolder:   isFolder,
			ModifiedAt: modifiedTime,
		})
	}

	return items, nil
}

func (w *WebDAVProvider) CreateFolder(physicalParentID string, name string) (string, error) {
	parent := physicalParentID
	if parent == "root" || parent == "" {
		parent = "/"
	}
	folderPath := path.Join(parent, name)

	resp, err := w.doRequest("MKCOL", folderPath, nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMethodNotAllowed {
		return "", fmt.Errorf("failed to create folder on WebDAV: status %d", resp.StatusCode)
	}

	return folderPath, nil
}

func (w *WebDAVProvider) UploadFile(physicalParentID string, filename string, r io.Reader, size int64) (string, error) {
	parent := physicalParentID
	if parent == "root" || parent == "" {
		parent = "/"
	}
	filePath := path.Join(parent, filename)

	headers := map[string]string{
		"Content-Type": "application/octet-stream",
	}

	resp, err := w.doRequest("PUT", filePath, r, headers)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return "", fmt.Errorf("webdav upload failed with status %d", resp.StatusCode)
	}

	return filePath, nil
}

func (w *WebDAVProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	resp, err := w.doRequest("GET", physicalID, nil, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("webdav download failed with status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (w *WebDAVProvider) DeleteFile(physicalID string) error {
	resp, err := w.doRequest("DELETE", physicalID, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("webdav delete failed with status %d", resp.StatusCode)
	}

	return nil
}

func (w *WebDAVProvider) RenameFile(physicalID string, newName string) error {
	parentDir := path.Dir(physicalID)
	newPath := path.Join(parentDir, newName)

	// Build absolute destination URI header
	u, err := url.Parse(w.BaseURL)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, newPath)

	headers := map[string]string{
		"Destination": u.String(),
		"Overwrite":   "T",
	}

	resp, err := w.doRequest("MOVE", physicalID, nil, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("webdav rename failed with status %d", resp.StatusCode)
	}

	return nil
}

func (w *WebDAVProvider) GetOAuthURL() string {
	return ""
}
