package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

type YandexProvider struct {
	clientID     string
	clientSecret string
	token        *oauth2.Token
	client       *http.Client
	onRefresh    func(*oauth2.Token)
}

func NewYandexProvider(clientID, clientSecret string, token *oauth2.Token, onRefresh func(*oauth2.Token)) *YandexProvider {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://oauth.yandex.com/authorize",
			TokenURL: "https://oauth.yandex.com/token",
		},
	}

	ctx := context.Background()
	ts := config.TokenSource(ctx, token)
	client := oauth2.NewClient(ctx, ts)

	refresherTransport := &tokenRefreshTransport{
		base:      client.Transport,
		source:    ts,
		onRefresh: onRefresh,
		lastToken: token.AccessToken,
	}

	return &YandexProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		token:        token,
		client:       &http.Client{Transport: refresherTransport, Timeout: 30 * time.Second},
		onRefresh:    onRefresh,
	}
}

type yandexDiskResponse struct {
	TotalSpace int64 `json:"total_space"`
	UsedSpace  int64 `json:"used_space"`
	User       struct {
		Login       string `json:"login"`
		DisplayName string `json:"display_name"`
	} `json:"user"`
}

func (y *YandexProvider) GetUserInfo() (name, email string, err error) {
	resp, err := y.client.Get("https://cloud-api.yandex.net/v1/disk")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to fetch Yandex Disk user info: %d", resp.StatusCode)
	}

	var data yandexDiskResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", err
	}

	return data.User.DisplayName, data.User.Login, nil
}

func (y *YandexProvider) GetQuota() (used, total int64, err error) {
	resp, err := y.client.Get("https://cloud-api.yandex.net/v1/disk")
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("failed to fetch Yandex Disk quota: %d", resp.StatusCode)
	}

	var data yandexDiskResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, 0, err
	}

	return data.UsedSpace, data.TotalSpace, nil
}

type yandexResource struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // "dir" or "file"
	Size int64  `json:"size"`
}

type yandexListResponse struct {
	Embedded struct {
		Items []yandexResource `json:"items"`
	} `json:"_embedded"`
}

func (y *YandexProvider) ListDirectory(remoteParentID string) ([]FileMetadata, error) {
	dirPath := remoteParentID
	if dirPath == "root" || dirPath == "" {
		dirPath = "disk:/"
	}

	u := fmt.Sprintf("https://cloud-api.yandex.net/v1/disk/resources?path=%s&limit=1000", url.QueryEscape(dirPath))
	resp, err := y.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // directory does not exist yet
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list Yandex directory: %d - %s", resp.StatusCode, string(respBody))
	}

	var data yandexListResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var items []FileMetadata
	for _, item := range data.Embedded.Items {
		items = append(items, FileMetadata{
			PhysicalID: item.Path,
			Name:       item.Name,
			Size:       item.Size,
			IsFolder:   item.Type == "dir",
			ModifiedAt: time.Now(),
		})
	}

	return items, nil
}

func (y *YandexProvider) CreateFolder(remoteParentID string, name string) (string, error) {
	parent := remoteParentID
	if parent == "root" || parent == "" {
		parent = "disk:/"
	}
	// Append folder name to parent path
	folderPath := parent
	if !strings.HasSuffix(folderPath, "/") {
		folderPath += "/"
	}
	folderPath += name

	u := fmt.Sprintf("https://cloud-api.yandex.net/v1/disk/resources?path=%s", url.QueryEscape(folderPath))
	req, err := http.NewRequest("PUT", u, nil)
	if err != nil {
		return "", err
	}

	resp, err := y.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 201 Created or 409 Conflict (if already exists)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create Yandex folder: %d - %s", resp.StatusCode, string(respBody))
	}

	return folderPath, nil
}

func (y *YandexProvider) UploadFile(remoteParentID string, filename string, r io.Reader, size int64) (string, error) {
	parent := remoteParentID
	if parent == "root" || parent == "" {
		parent = "disk:/"
	}
	filePath := parent
	if !strings.HasSuffix(filePath, "/") {
		filePath += "/"
	}
	filePath += filename

	// 1. Get upload link
	u := fmt.Sprintf("https://cloud-api.yandex.net/v1/disk/resources/upload?path=%s&overwrite=true", url.QueryEscape(filePath))
	resp, err := y.client.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get Yandex upload link: %d - %s", resp.StatusCode, string(respBody))
	}

	var link struct {
		Href string `json:"href"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&link); err != nil {
		return "", err
	}

	// 2. Upload file to Yandex via the upload link using PUT
	req, err := http.NewRequest("PUT", link.Href, r)
	if err != nil {
		return "", err
	}
	if size > 0 {
		req.ContentLength = size
	}

	uploadResp, err := y.client.Do(req)
	if err != nil {
		return "", err
	}
	defer uploadResp.Body.Close()

	if uploadResp.StatusCode != http.StatusCreated && uploadResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(uploadResp.Body)
		return "", fmt.Errorf("failed to upload data to Yandex: %d - %s", uploadResp.StatusCode, string(respBody))
	}

	return filePath, nil
}

func (y *YandexProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	// 1. Get download link
	u := fmt.Sprintf("https://cloud-api.yandex.net/v1/disk/resources/download?path=%s", url.QueryEscape(physicalID))
	resp, err := y.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get Yandex download link, status: %d", resp.StatusCode)
	}

	var link struct {
		Href string `json:"href"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&link); err != nil {
		return nil, err
	}

	// 2. Stream download content
	contentResp, err := y.client.Get(link.Href)
	if err != nil {
		return nil, err
	}

	if contentResp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, contentResp.Body)
		contentResp.Body.Close()
		return nil, fmt.Errorf("failed to download content from Yandex link: %d", contentResp.StatusCode)
	}

	return contentResp.Body, nil
}

func (y *YandexProvider) DeleteFile(physicalID string) error {
	u := fmt.Sprintf("https://cloud-api.yandex.net/v1/disk/resources?path=%s", url.QueryEscape(physicalID))
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	resp, err := y.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete Yandex resource: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (y *YandexProvider) RenameFile(physicalID string, newName string) error {
	parent := path.Dir(physicalID)
	// Make sure Yandex disk prefix matches
	if !strings.HasPrefix(parent, "disk:") {
		parent = "disk:" + parent
	}
	newPath := parent
	if !strings.HasSuffix(newPath, "/") {
		newPath += "/"
	}
	newPath += newName

	u := fmt.Sprintf("https://cloud-api.yandex.net/v1/disk/resources/move?from=%s&path=%s&overwrite=true",
		url.QueryEscape(physicalID), url.QueryEscape(newPath))

	resp, err := y.client.Post(u, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to rename Yandex resource: %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (y *YandexProvider) GetOAuthURL() string {
	return ""
}
