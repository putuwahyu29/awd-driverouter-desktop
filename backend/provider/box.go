package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

type BoxProvider struct {
	clientID     string
	clientSecret string
	token        *oauth2.Token
	client       *http.Client
	onRefresh    func(*oauth2.Token)
}

func NewBoxProvider(clientID, clientSecret string, token *oauth2.Token, onRefresh func(*oauth2.Token)) *BoxProvider {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://account.box.com/api/oauth2/authorize",
			TokenURL: "https://api.box.com/oauth2/token",
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

	return &BoxProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		token:        token,
		client:       &http.Client{Transport: refresherTransport, Timeout: 30 * time.Second},
		onRefresh:    onRefresh,
	}
}

type boxUserResponse struct {
	Name  string `json:"name"`
	Login string `json:"login"`
}

func (b *BoxProvider) GetUserInfo() (name, email string, err error) {
	resp, err := b.client.Get("https://api.box.com/2.0/users/me")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to get Box user info, status: %d", resp.StatusCode)
	}

	var user boxUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", "", err
	}

	return user.Name, user.Login, nil
}

type boxSpaceResponse struct {
	SpaceAmount int64 `json:"space_amount"`
	SpaceUsed   int64 `json:"space_used"`
}

func (b *BoxProvider) GetQuota() (used, total int64, err error) {
	resp, err := b.client.Get("https://api.box.com/2.0/users/me")
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("failed to get Box quota, status: %d", resp.StatusCode)
	}

	var quota boxSpaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&quota); err != nil {
		return 0, 0, err
	}

	return quota.SpaceUsed, quota.SpaceAmount, nil
}

type boxItem struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "folder" or "file"
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type boxFolderItemsResponse struct {
	Entries []boxItem `json:"entries"`
}

func (b *BoxProvider) ListDirectory(remoteParentID string) ([]FileMetadata, error) {
	folderID := remoteParentID
	if folderID == "root" || folderID == "" {
		folderID = "0" // Box root folder ID is "0"
	}

	resp, err := b.client.Get(fmt.Sprintf("https://api.box.com/2.0/folders/%s/items?fields=id,type,name,size", folderID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list Box directory, status: %d", resp.StatusCode)
	}

	var list boxFolderItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}

	var items []FileMetadata
	for _, entry := range list.Entries {
		items = append(items, FileMetadata{
			PhysicalID: entry.ID,
			Name:       entry.Name,
			Size:       entry.Size,
			IsFolder:   entry.Type == "folder",
			ModifiedAt: time.Now(),
		})
	}

	return items, nil
}

func (b *BoxProvider) CreateFolder(remoteParentID string, name string) (string, error) {
	parentID := remoteParentID
	if parentID == "root" || parentID == "" {
		parentID = "0"
	}

	payload := map[string]interface{}{
		"name": name,
		"parent": map[string]string{
			"id": parentID,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := b.client.Post("https://api.box.com/2.0/folders", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create Box folder, status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var folder boxItem
	if err := json.NewDecoder(resp.Body).Decode(&folder); err != nil {
		return "", err
	}

	return folder.ID, nil
}

func (b *BoxProvider) UploadFile(remoteParentID string, filename string, r io.Reader, size int64) (string, error) {
	parentID := remoteParentID
	if parentID == "root" || parentID == "" {
		parentID = "0"
	}

	// Box upload endpoint is https://upload.box.com/api/2.0/files/content
	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)

	// Box requires metadata attributes in a separate form field
	attributes := map[string]interface{}{
		"name": filename,
		"parent": map[string]string{
			"id": parentID,
		},
	}

	attrBytes, err := json.Marshal(attributes)
	if err != nil {
		return "", err
	}

	err = writer.WriteField("attributes", string(attrBytes))
	if err != nil {
		return "", err
	}

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(part, r)
	if err != nil {
		return "", err
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://upload.box.com/api/2.0/files/content", bodyBuf)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := b.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload file to Box, status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Box response lists uploaded entries
	type uploadResponse struct {
		Entries []boxItem `json:"entries"`
	}

	var res uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if len(res.Entries) == 0 {
		return "", fmt.Errorf("no uploaded files found in Box response")
	}

	return res.Entries[0].ID, nil
}

func (b *BoxProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	// GET /files/{file_id}/content redirects to actual content location
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.box.com/2.0/files/%s/content", physicalID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download from Box, status: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (b *BoxProvider) DeleteFile(physicalID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://api.box.com/2.0/files/%s", physicalID), nil)
	if err != nil {
		return err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete from Box, status: %d", resp.StatusCode)
	}

	return nil
}

func (b *BoxProvider) RenameFile(physicalID string, newName string) error {
	payload := map[string]string{
		"name": newName,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("https://api.box.com/2.0/files/%s", physicalID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to rename on Box, status: %d", resp.StatusCode)
	}

	return nil
}

func (b *BoxProvider) GetOAuthURL() string {
	return ""
}
