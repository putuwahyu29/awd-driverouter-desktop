package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/oauth2"
)

type PCloudProvider struct {
	clientID     string
	clientSecret string
	token        *oauth2.Token
	client       *http.Client
	onRefresh    func(*oauth2.Token)
}

func NewPCloudProvider(clientID, clientSecret string, token *oauth2.Token, onRefresh func(*oauth2.Token)) *PCloudProvider {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://my.pcloud.com/oauth2/authorize",
			TokenURL: "https://api.pcloud.com/oauth2_token",
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

	return &PCloudProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		token:        token,
		client:       &http.Client{Transport: refresherTransport, Timeout: 30 * time.Second},
		onRefresh:    onRefresh,
	}
}

type pcloudUserResponse struct {
	Result    int    `json:"result"`
	Error     string `json:"error"`
	Email     string `json:"email"`
	Registered bool   `json:"registered"`
	Quota     int64  `json:"quota"`
	UsedQuota int64  `json:"usedquota"`
}

func (p *PCloudProvider) GetUserInfo() (name, email string, err error) {
	resp, err := p.client.Get("https://api.pcloud.com/userinfo")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to get pCloud user info: %d", resp.StatusCode)
	}

	var data pcloudUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", err
	}

	if data.Result != 0 {
		return "", "", fmt.Errorf("pCloud error: %s", data.Error)
	}

	return data.Email, data.Email, nil
}

func (p *PCloudProvider) GetQuota() (used, total int64, err error) {
	resp, err := p.client.Get("https://api.pcloud.com/userinfo")
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("failed to get pCloud quota: %d", resp.StatusCode)
	}

	var data pcloudUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, 0, err
	}

	if data.Result != 0 {
		return 0, 0, fmt.Errorf("pCloud error: %s", data.Error)
	}

	return data.UsedQuota, data.Quota, nil
}

type pcloudItem struct {
	Name     string `json:"name"`
	IsFolder bool   `json:"isfolder"`
	FileID   int64  `json:"fileid"`
	FolderID int64  `json:"folderid"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

type pcloudFolderResponse struct {
	Result int          `json:"result"`
	Error  string       `json:"error"`
	Metadata struct {
		Contents []pcloudItem `json:"contents"`
	} `json:"metadata"`
}

func (p *PCloudProvider) ListDirectory(remoteParentID string) ([]FileMetadata, error) {
	folderID := remoteParentID
	if folderID == "root" || folderID == "" {
		folderID = "0"
	}

	u := fmt.Sprintf("https://api.pcloud.com/listfolder?folderid=%s", folderID)
	resp, err := p.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list pCloud folder, status: %d", resp.StatusCode)
	}

	var data pcloudFolderResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data.Result != 0 {
		return nil, fmt.Errorf("pCloud error: %s", data.Error)
	}

	var items []FileMetadata
	for _, entry := range data.Metadata.Contents {
		physID := ""
		if entry.IsFolder {
			physID = strconv.FormatInt(entry.FolderID, 10)
		} else {
			physID = strconv.FormatInt(entry.FileID, 10)
		}

		var modTime time.Time
		if entry.Modified != "" {
			modTime, _ = time.Parse(time.RFC1123, entry.Modified)
		}

		items = append(items, FileMetadata{
			PhysicalID: physID,
			Name:       entry.Name,
			Size:       entry.Size,
			IsFolder:   entry.IsFolder,
			ModifiedAt: modTime,
		})
	}

	return items, nil
}

func (p *PCloudProvider) CreateFolder(remoteParentID string, name string) (string, error) {
	parentID := remoteParentID
	if parentID == "root" || parentID == "" {
		parentID = "0"
	}

	u := fmt.Sprintf("https://api.pcloud.com/createfolder?name=%s&folderid=%s", url.QueryEscape(name), parentID)
	resp, err := p.client.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create pCloud folder: %d", resp.StatusCode)
	}

	var data struct {
		Result   int    `json:"result"`
		Error    string `json:"error"`
		Metadata struct {
			FolderID int64 `json:"folderid"`
		} `json:"metadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if data.Result != 0 {
		return "", fmt.Errorf("pCloud error: %s", data.Error)
	}

	return strconv.FormatInt(data.Metadata.FolderID, 10), nil
}

func (p *PCloudProvider) UploadFile(remoteParentID string, filename string, r io.Reader, size int64) (string, error) {
	parentID := remoteParentID
	if parentID == "root" || parentID == "" {
		parentID = "0"
	}

	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)

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

	u := fmt.Sprintf("https://api.pcloud.com/uploadfile?folderid=%s", parentID)
	req, err := http.NewRequest("POST", u, bodyBuf)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload to pCloud: %d - %s", resp.StatusCode, string(respBody))
	}

	var data struct {
		Result   int    `json:"result"`
		Error    string `json:"error"`
		FileIDs  []int64 `json:"fileids"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if data.Result != 0 {
		return "", fmt.Errorf("pCloud error: %s", data.Error)
	}

	if len(data.FileIDs) == 0 {
		return "", fmt.Errorf("no uploaded file IDs returned by pCloud")
	}

	return strconv.FormatInt(data.FileIDs[0], 10), nil
}

func (p *PCloudProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	// 1. Get file download link
	u := fmt.Sprintf("https://api.pcloud.com/getfilelink?fileid=%s", physicalID)
	resp, err := p.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get pCloud file link: %d", resp.StatusCode)
	}

	var data struct {
		Result int      `json:"result"`
		Error  string   `json:"error"`
		Path   string   `json:"path"`
		Hosts  []string `json:"hosts"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data.Result != 0 {
		return nil, fmt.Errorf("pCloud error: %s", data.Error)
	}

	if len(data.Hosts) == 0 {
		return nil, fmt.Errorf("no pCloud download hosts returned")
	}

	downloadURL := fmt.Sprintf("https://%s%s", data.Hosts[0], data.Path)

	// 2. Stream content
	contentResp, err := p.client.Get(downloadURL)
	if err != nil {
		return nil, err
	}

	if contentResp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, contentResp.Body)
		contentResp.Body.Close()
		return nil, fmt.Errorf("failed to download from pCloud link, status: %d", contentResp.StatusCode)
	}

	return contentResp.Body, nil
}

func (p *PCloudProvider) DeleteFile(physicalID string) error {
	// Standard delete command
	u := fmt.Sprintf("https://api.pcloud.com/deletefile?fileid=%s", physicalID)
	resp, err := p.client.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete pCloud file: %d", resp.StatusCode)
	}

	var data struct {
		Result int    `json:"result"`
		Error  string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	if data.Result != 0 {
		return fmt.Errorf("pCloud error: %s", data.Error)
	}

	return nil
}

func (p *PCloudProvider) RenameFile(physicalID string, newName string) error {
	u := fmt.Sprintf("https://api.pcloud.com/renamefile?fileid=%s&toname=%s", physicalID, url.QueryEscape(newName))
	resp, err := p.client.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to rename pCloud file: %d", resp.StatusCode)
	}

	var data struct {
		Result int    `json:"result"`
		Error  string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	if data.Result != 0 {
		return fmt.Errorf("pCloud error: %s", data.Error)
	}

	return nil
}

func (p *PCloudProvider) GetOAuthURL() string {
	return ""
}
