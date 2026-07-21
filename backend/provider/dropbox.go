package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

type DropboxProvider struct {
	Config *oauth2.Config
	Client *http.Client
	Token  *oauth2.Token
	OnTokenRefresh func(*oauth2.Token)
}

func NewDropboxProvider(clientID, clientSecret string, token *oauth2.Token, onRefresh func(*oauth2.Token)) *DropboxProvider {
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://www.dropbox.com/oauth2/authorize",
			TokenURL: "https://api.dropboxapi.com/oauth2/token",
		},
		RedirectURL: "http://localhost:5998/oauth/callback",
	}

	ts := conf.TokenSource(oauth2.NoContext, token)
	client := oauth2.NewClient(oauth2.NoContext, ts)

	dbx := &DropboxProvider{
		Config:         conf,
		Token:          token,
		OnTokenRefresh: onRefresh,
	}

	dbx.Client = &http.Client{
		Transport: &tokenRefreshTransport{
			base:   client.Transport,
			source: ts,
			onRefresh: func(newToken *oauth2.Token) {
				dbx.Token = newToken
				if dbx.OnTokenRefresh != nil {
					dbx.OnTokenRefresh(newToken)
				}
			},
		},
	}

	return dbx
}

func (d *DropboxProvider) GetUserInfo() (name, email string, err error) {
	resp, err := d.Client.Post("https://api.dropboxapi.com/2/users/get_current_account", "application/json", bytes.NewBuffer([]byte("null")))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("userinfo request failed (%d): %s", resp.StatusCode, string(body))
	}

	var data struct {
		Name struct {
			DisplayName string `json:"display_name"`
		} `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", err
	}
	return data.Name.DisplayName, data.Email, nil
}

func (d *DropboxProvider) GetQuota() (used, total int64, err error) {
	resp, err := d.Client.Post("https://api.dropboxapi.com/2/users/get_space_usage", "application/json", bytes.NewBuffer([]byte("null")))
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, 0, fmt.Errorf("space usage request failed (%d): %s", resp.StatusCode, string(body))
	}

	var data struct {
		Used       int64 `json:"used"`
		Allocation struct {
			Tag       string `json:".tag"`
			Allocated int64  `json:"allocated"`
		} `json:"allocation"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, 0, err
	}

	return data.Used, data.Allocation.Allocated, nil
}

func (d *DropboxProvider) ListDirectory(remoteParentID string) ([]FileMetadata, error) {
	// For Dropbox, listing a folder uses the path (e.g. "" for root or "/driverouter" or the file ID "id:...")
	path := remoteParentID
	if path == "root" {
		path = ""
	}

	var list []FileMetadata
	var cursor string
	var hasMore bool

	// Initial call
	reqData := map[string]interface{}{
		"path": path,
	}
	bodyBytes, _ := json.Marshal(reqData)

	resp, err := d.Client.Post("https://api.dropboxapi.com/2/files/list_folder", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	for {
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("list directory failed (%d): %s", resp.StatusCode, string(body))
		}

		var result struct {
			Entries []struct {
				Tag            string `json:".tag"` // "file" or "folder"
				Name           string `json:"name"`
				ID             string `json:"id"`
				PathLower      string `json:"path_lower"`
				Size           int64  `json:"size,omitempty"`
				ServerModified string `json:"server_modified,omitempty"`
			} `json:"entries"`
			Cursor  string `json:"cursor"`
			HasMore bool   `json:"has_more"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		for _, entry := range result.Entries {
			isFolder := entry.Tag == "folder"
			var modified time.Time
			if entry.ServerModified != "" {
				modified, _ = time.Parse(time.RFC3339, entry.ServerModified)
			} else {
				modified = time.Now()
			}

			list = append(list, FileMetadata{
				PhysicalID: entry.ID,
				Name:       entry.Name,
				Size:       entry.Size,
				IsFolder:   isFolder,
				Provider:   "dropbox",
				CreatedAt:  modified, // Dropbox doesn't store explicit created time, default to modified
				ModifiedAt: modified,
			})
		}

		cursor = result.Cursor
		hasMore = result.HasMore

		if !hasMore {
			break
		}

		// Continue call
		reqData = map[string]interface{}{
			"cursor": cursor,
		}
		bodyBytes, _ = json.Marshal(reqData)
		resp, err = d.Client.Post("https://api.dropboxapi.com/2/files/list_folder/continue", "application/json", bytes.NewBuffer(bodyBytes))
		if err != nil {
			return nil, err
		}
	}

	return list, nil
}

func (d *DropboxProvider) CreateFolder(remoteParentID string, name string) (physicalID string, err error) {
	// Dropbox expects paths like "/driverouter/foldername"
	// To simplify, if remoteParentID is a path, we append.
	// If it is an ID (e.g. "id:...") we might need to resolve it, but we can manage our folders inside a dedicated /driverouter prefix.
	// Let's resolve the parent path. For dropbox, we can just specify a path directory structure in SQLite and create it.
	// If remoteParentID starts with "id:", we can query metadata first or assume we are building absolute paths.
	// Let's query the metadata of the parent ID to get its path_display, then append name.
	parentPath := ""
	if remoteParentID != "" && remoteParentID != "root" {
		path, err := d.getPathForID(remoteParentID)
		if err == nil {
			parentPath = path
		} else {
			parentPath = ""
		}
	} else {
		parentPath = ""
	}

	folderPath := parentPath + "/" + name

	reqData := map[string]interface{}{
		"path":       folderPath,
		"autorename": false,
	}
	bodyBytes, _ := json.Marshal(reqData)

	resp, err := d.Client.Post("https://api.dropboxapi.com/2/files/create_folder_v2", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create folder failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Metadata struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Metadata.ID, nil
}

func (d *DropboxProvider) UploadFile(remoteParentID string, filename string, r io.Reader, size int64) (physicalID string, err error) {
	parentPath := ""
	if remoteParentID != "" && remoteParentID != "root" {
		path, err := d.getPathForID(remoteParentID)
		if err == nil {
			parentPath = path
		} else {
			parentPath = ""
		}
	} else {
		parentPath = ""
	}

	filePath := parentPath + "/" + filename

	arg := map[string]interface{}{
		"path":       filePath,
		"mode":       "overwrite",
		"autorename": false,
		"mute":       false,
	}
	argBytes, _ := json.Marshal(arg)

	req, err := http.NewRequest("POST", "https://content.dropboxapi.com/2/files/upload", r)
	if err != nil {
		return "", err
	}
	req.Header.Set("Dropbox-API-Arg", string(argBytes))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := d.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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

func (d *DropboxProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	arg := map[string]interface{}{
		"path": physicalID,
	}
	argBytes, _ := json.Marshal(arg)

	req, err := http.NewRequest("POST", "https://content.dropboxapi.com/2/files/download", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Dropbox-API-Arg", string(argBytes))

	resp, err := d.Client.Do(req)
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

func (d *DropboxProvider) DeleteFile(physicalID string) error {
	reqData := map[string]interface{}{
		"path": physicalID,
	}
	bodyBytes, _ := json.Marshal(reqData)

	resp, err := d.Client.Post("https://api.dropboxapi.com/2/files/delete_v2", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (d *DropboxProvider) getPathForID(physicalID string) (string, error) {
	reqData := map[string]interface{}{
		"path": physicalID,
	}
	bodyBytes, _ := json.Marshal(reqData)

	resp, err := d.Client.Post("https://api.dropboxapi.com/2/files/get_metadata", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("get_metadata failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		PathDisplay string `json:"path_display"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.PathDisplay, nil
}
