package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/oauth2"
)

type GoogleDriveProvider struct {
	Config *oauth2.Config
	Client *http.Client
	Token  *oauth2.Token
	OnTokenRefresh func(*oauth2.Token)
}

func NewGoogleDriveProvider(clientID, clientSecret string, token *oauth2.Token, onRefresh func(*oauth2.Token)) *GoogleDriveProvider {
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		RedirectURL: "http://localhost:5998/oauth/callback",
		Scopes:      []string{"https://www.googleapis.com/auth/drive", "https://www.googleapis.com/auth/userinfo.profile", "https://www.googleapis.com/auth/userinfo.email"},
	}

	// Create background token source that auto refreshes
	ts := conf.TokenSource(oauth2.NoContext, token)
	client := oauth2.NewClient(oauth2.NoContext, ts)

	// Custom HTTP client that will intercept refreshed tokens
	g := &GoogleDriveProvider{
		Config:         conf,
		Token:          token,
		OnTokenRefresh: onRefresh,
	}

	// Wrap the transport so we can detect token refreshes
	g.Client = &http.Client{
		Transport: &tokenRefreshTransport{
			base:   client.Transport,
			source: ts,
			onRefresh: func(newToken *oauth2.Token) {
				g.Token = newToken
				if g.OnTokenRefresh != nil {
					g.OnTokenRefresh(newToken)
				}
			},
		},
	}

	return g
}

type tokenRefreshTransport struct {
	base      http.RoundTripper
	source    oauth2.TokenSource
	onRefresh func(*oauth2.Token)
	lastToken string
}

func (t *tokenRefreshTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tok, err := t.source.Token()
	if err == nil && tok.AccessToken != t.lastToken {
		t.lastToken = tok.AccessToken
		if t.onRefresh != nil {
			t.onRefresh(tok)
		}
	}
	return t.base.RoundTrip(req)
}

func (g *GoogleDriveProvider) GetUserInfo() (name, email string, err error) {
	resp, err := g.Client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return "", "", fmt.Errorf("userinfo request failed with status %d", resp.StatusCode)
	}

	var data struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", err
	}
	return data.Name, data.Email, nil
}

func (g *GoogleDriveProvider) GetQuota() (used, total int64, err error) {
	resp, err := g.Client.Get("https://www.googleapis.com/drive/v3/about?fields=storageQuota")
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return 0, 0, fmt.Errorf("about request failed with status %d", resp.StatusCode)
	}

	var data struct {
		StorageQuota struct {
			Limit string `json:"limit"`
			Usage string `json:"usage"`
		} `json:"storageQuota"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, 0, err
	}

	limit, _ := strconv.ParseInt(data.StorageQuota.Limit, 10, 64)
	usage, _ := strconv.ParseInt(data.StorageQuota.Usage, 10, 64)

	// If Google accounts have "unlimited" quota, limit is returned as 0 or empty, default to 15GB if zero
	if limit == 0 {
		limit = 15 * 1024 * 1024 * 1024
	}

	return usage, limit, nil
}

func (g *GoogleDriveProvider) ListDirectory(remoteParentID string) ([]FileMetadata, error) {
	var list []FileMetadata
	pageToken := ""

	for {
		q := fmt.Sprintf("'%s' in parents and trashed = false", remoteParentID)
		fields := "nextPageToken,files(id, name, size, mimeType, createdTime, modifiedTime, shared)"
		urlStr := fmt.Sprintf("https://www.googleapis.com/drive/v3/files?q=%s&fields=%s&pageSize=1000", url.QueryEscape(q), url.QueryEscape(fields))
		if pageToken != "" {
			urlStr += "&pageToken=" + url.QueryEscape(pageToken)
		}

		resp, err := g.Client.Get(urlStr)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("list directory failed (%d): %s", resp.StatusCode, string(body))
		}

		var result struct {
			NextPageToken string `json:"nextPageToken"`
			Files         []struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				Size         string `json:"size"`
				MimeType     string `json:"mimeType"`
				CreatedTime  string `json:"createdTime"`
				ModifiedTime string `json:"modifiedTime"`
				Shared       bool   `json:"shared"`
			} `json:"files"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		for _, f := range result.Files {
			isFolder := f.MimeType == "application/vnd.google-apps.folder"
			size, _ := strconv.ParseInt(f.Size, 10, 64)

			created, _ := time.Parse(time.RFC3339, f.CreatedTime)
			modified, _ := time.Parse(time.RFC3339, f.ModifiedTime)

			list = append(list, FileMetadata{
				PhysicalID: f.ID,
				Name:       f.Name,
				Size:       size,
				IsFolder:   isFolder,
				Provider:   "google",
				CreatedAt:  created,
				ModifiedAt: modified,
				Shared:     f.Shared,
			})
		}

		pageToken = result.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return list, nil
}

func (g *GoogleDriveProvider) CreateFolder(remoteParentID string, name string) (physicalID string, err error) {
	meta := map[string]interface{}{
		"name":     name,
		"mimeType": "application/vnd.google-apps.folder",
	}
	if remoteParentID != "" && remoteParentID != "root" {
		meta["parents"] = []string{remoteParentID}
	}

	bodyBytes, _ := json.Marshal(meta)
	resp, err := g.Client.Post("https://www.googleapis.com/drive/v3/files", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create folder failed: %s", string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

func (g *GoogleDriveProvider) UploadFile(remoteParentID string, filename string, r io.Reader, size int64) (physicalID string, err error) {
	// If the file is small (< 5MB), use standard multipart upload to avoid network round-trip overhead.
	const maxSimpleUpload = 5 * 1024 * 1024 // 5 MB

	if size <= maxSimpleUpload {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// 1. Metadata Part
		metadataHeader := make(textproto.MIMEHeader)
		metadataHeader.Set("Content-Type", "application/json; charset=UTF-8")
		metaWriter, err := writer.CreatePart(metadataHeader)
		if err != nil {
			return "", err
		}

		meta := map[string]interface{}{
			"name": filename,
		}
		if remoteParentID != "" && remoteParentID != "root" {
			meta["parents"] = []string{remoteParentID}
		}
		metaBytes, _ := json.Marshal(meta)
		_, _ = metaWriter.Write(metaBytes)

		// 2. Media Part
		mediaHeader := make(textproto.MIMEHeader)
		mediaHeader.Set("Content-Type", "application/octet-stream")
		mediaWriter, err := writer.CreatePart(mediaHeader)
		if err != nil {
			return "", err
		}

		_, err = io.Copy(mediaWriter, r)
		if err != nil {
			return "", err
		}

		writer.Close()

		req, err := http.NewRequest("POST", "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart", &buf)
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "multipart/related; boundary="+writer.Boundary())

		resp, err := g.Client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("upload failed (%d): %s", resp.StatusCode, string(body))
		}

		var result struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", err
		}

		return result.ID, nil
	}

	// Large resumable upload session path
	// 1. Initiate Resumable Upload
	meta := map[string]interface{}{
		"name": filename,
	}
	if remoteParentID != "" && remoteParentID != "root" {
		meta["parents"] = []string{remoteParentID}
	}
	metaBytes, _ := json.Marshal(meta)

	initURL := "https://www.googleapis.com/upload/drive/v3/files?uploadType=resumable"
	req, err := http.NewRequest("POST", initURL, bytes.NewBuffer(metaBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("X-Upload-Content-Type", "application/octet-stream")
	req.Header.Set("X-Upload-Content-Length", strconv.FormatInt(size, 10))

	resp, err := g.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to initiate resumable upload session (%d): %s", resp.StatusCode, string(body))
	}

	uploadURL := resp.Header.Get("Location")
	if uploadURL == "" {
		return "", fmt.Errorf("resumable upload session initiated but Location header is missing")
	}

	// 2. Upload chunks in multiples of 256 KB (e.g. 1 MB chunks)
	const chunkSize = 256 * 1024 * 4 // 1 MB
	buf := make([]byte, chunkSize)
	var bytesUploaded int64 = 0

	for {
		n, readErr := io.ReadFull(r, buf)
		if n > 0 {
			chunkRange := fmt.Sprintf("bytes %d-%d/%d", bytesUploaded, bytesUploaded+int64(n)-1, size)
			reqPut, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(buf[:n]))
			if err != nil {
				return "", err
			}
			reqPut.Header.Set("Content-Length", strconv.Itoa(n))
			reqPut.Header.Set("Content-Range", chunkRange)

			respPut, err := g.Client.Do(reqPut)
			if err != nil {
				return "", err
			}
			defer respPut.Body.Close()

			if respPut.StatusCode != http.StatusOK && respPut.StatusCode != http.StatusCreated && respPut.StatusCode != http.StatusPermanentRedirect {
				body, _ := io.ReadAll(respPut.Body)
				return "", fmt.Errorf("chunk upload failed (%d) at range %s: %s", respPut.StatusCode, chunkRange, string(body))
			}

			bytesUploaded += int64(n)

			if respPut.StatusCode == http.StatusOK || respPut.StatusCode == http.StatusCreated {
				var result struct {
					ID string `json:"id"`
				}
				if err := json.NewDecoder(respPut.Body).Decode(&result); err != nil {
					return "", err
				}
				return result.ID, nil
			}
		}

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		} else if readErr != nil {
			return "", readErr
		}
	}

	return "", fmt.Errorf("upload completed but final response was not received")
}

func (g *GoogleDriveProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	urlStr := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?alt=media", physicalID)
	resp, err := g.Client.Get(urlStr)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (g *GoogleDriveProvider) DeleteFile(physicalID string) error {
	payload := `{"trashed": true}`
	req, err := http.NewRequest("PATCH", "https://www.googleapis.com/drive/v3/files/"+physicalID, bytes.NewBufferString(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (g *GoogleDriveProvider) GetWebURL(physicalID string, isFolder bool) (string, error) {
	// Set permission to anyone:reader to ensure it is publicly shareable by link
	permURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s/permissions", url.PathEscape(physicalID))
	permBody := []byte(`{"role":"reader","type":"anyone"}`)
	req, err := http.NewRequest("POST", permURL, bytes.NewBuffer(permBody))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		respPerm, errPerm := g.Client.Do(req)
		if errPerm == nil {
			respPerm.Body.Close()
		}
	}

	urlStr := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?fields=webViewLink", url.PathEscape(physicalID))
	resp, err := g.Client.Get(urlStr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("get web link failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		WebViewLink string `json:"webViewLink"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.WebViewLink, nil
}

func (g *GoogleDriveProvider) ListPermissions(physicalID string) ([]SharePermission, error) {
	urlStr := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s/permissions?fields=permissions(id,type,role,emailAddress,displayName)", url.PathEscape(physicalID))
	resp, err := g.Client.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list permissions failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Permissions []struct {
			ID           string `json:"id"`
			Type         string `json:"type"`
			Role         string `json:"role"`
			EmailAddress string `json:"emailAddress"`
			DisplayName  string `json:"displayName"`
		} `json:"permissions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var list []SharePermission
	for _, p := range result.Permissions {
		list = append(list, SharePermission{
			ID:           p.ID,
			Type:         p.Type,
			Role:         p.Role,
			EmailAddress: p.EmailAddress,
			DisplayName:  p.DisplayName,
		})
	}
	return list, nil
}

func (g *GoogleDriveProvider) AddPermission(physicalID string, email string, role string) error {
	urlStr := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s/permissions", url.PathEscape(physicalID))
	reqBody := map[string]string{
		"role":         role,
		"type":         "user",
		"emailAddress": email,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	resp, err := g.Client.Post(urlStr, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add permission failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (g *GoogleDriveProvider) DeletePermission(physicalID string, permID string) error {
	urlStr := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s/permissions/%s", url.PathEscape(physicalID), url.PathEscape(permID))
	req, err := http.NewRequest("DELETE", urlStr, nil)
	if err != nil {
		return err
	}

	resp, err := g.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete permission failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (g *GoogleDriveProvider) SetGeneralAccess(physicalID string, accessType string, role string) error {
	if accessType == "anyone" {
		urlStr := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s/permissions", url.PathEscape(physicalID))
		reqBody := map[string]string{
			"role": role,
			"type": "anyone",
		}
		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := g.Client.Post(urlStr, "application/json", bytes.NewBuffer(bodyBytes))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("set general access failed (%d): %s", resp.StatusCode, string(body))
		}
	} else {
		_ = g.DeletePermission(physicalID, "anyoneWithLink")
	}
	return nil
}

