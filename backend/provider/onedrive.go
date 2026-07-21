package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/oauth2"
)

type OneDriveProvider struct {
	Config *oauth2.Config
	Client *http.Client
	Token  *oauth2.Token
	OnTokenRefresh func(*oauth2.Token)
}

func NewOneDriveProvider(clientID, clientSecret string, token *oauth2.Token, onRefresh func(*oauth2.Token)) *OneDriveProvider {
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		},
		RedirectURL: "http://localhost:5998/oauth/callback",
		Scopes:      []string{"files.readwrite", "offline_access", "User.Read"},
	}

	ts := conf.TokenSource(oauth2.NoContext, token)
	client := oauth2.NewClient(oauth2.NoContext, ts)

	od := &OneDriveProvider{
		Config:         conf,
		Token:          token,
		OnTokenRefresh: onRefresh,
	}

	od.Client = &http.Client{
		Transport: &tokenRefreshTransport{
			base:   client.Transport,
			source: ts,
			onRefresh: func(newToken *oauth2.Token) {
				od.Token = newToken
				if od.OnTokenRefresh != nil {
					od.OnTokenRefresh(newToken)
				}
			},
		},
	}

	return od
}

func (od *OneDriveProvider) GetUserInfo() (name, email string, err error) {
	resp, err := od.Client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("userinfo request failed (%d): %s", resp.StatusCode, string(body))
	}

	var data struct {
		DisplayName       string `json:"displayName"`
		Mail              string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", err
	}

	emailAddr := data.Mail
	if emailAddr == "" {
		emailAddr = data.UserPrincipalName
	}
	return data.DisplayName, emailAddr, nil
}

func (od *OneDriveProvider) GetQuota() (used, total int64, err error) {
	resp, err := od.getWithRetry("https://graph.microsoft.com/v1.0/me/drive")
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, 0, fmt.Errorf("drive info request failed (%d): %s", resp.StatusCode, string(body))
	}

	var data struct {
		Quota struct {
			Deleted int64 `json:"deleted"`
			Remaining int64 `json:"remaining"`
			State string `json:"state"`
			Total int64 `json:"total"`
			Used int64 `json:"used"`
		} `json:"quota"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, 0, err
	}

	return data.Quota.Used, data.Quota.Total, nil
}

func (od *OneDriveProvider) ListDirectory(remoteParentID string) ([]FileMetadata, error) {
	var list []FileMetadata
	var urlStr string
	if remoteParentID == "" || remoteParentID == "root" {
		urlStr = "https://graph.microsoft.com/v1.0/me/drive/root/children"
	} else {
		urlStr = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s/children", remoteParentID)
	}

	for {
		resp, err := od.getWithRetry(urlStr)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("list directory failed (%d): %s", resp.StatusCode, string(body))
		}

		var result struct {
			NextLink string `json:"@odata.nextLink"`
			Value    []struct {
				ID                   string `json:"id"`
				Name                 string `json:"name"`
				Size                 int64  `json:"size"`
				Folder               *struct {
					ChildCount int `json:"childCount"`
				} `json:"folder"`
				CreatedDateTime      string `json:"createdDateTime"`
				LastModifiedDateTime string `json:"lastModifiedDateTime"`
				Shared               *struct {
					Scope string `json:"scope"`
				} `json:"shared"`
			} `json:"value"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		for _, f := range result.Value {
			isFolder := f.Folder != nil
			created, _ := time.Parse(time.RFC3339, f.CreatedDateTime)
			modified, _ := time.Parse(time.RFC3339, f.LastModifiedDateTime)
			isShared := f.Shared != nil

			list = append(list, FileMetadata{
				PhysicalID: f.ID,
				Name:       f.Name,
				Size:       f.Size,
				IsFolder:   isFolder,
				Provider:   "onedrive",
				CreatedAt:  created,
				ModifiedAt: modified,
				Shared:     isShared,
			})
		}

		urlStr = result.NextLink
		if urlStr == "" {
			break
		}
	}

	return list, nil
}

func (od *OneDriveProvider) getWithRetry(urlStr string) (*http.Response, error) {
	const maxAttempts = 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := od.Client.Get(urlStr)
		if err != nil {
			lastErr = err
			if attempt == maxAttempts {
				return nil, lastErr
			}
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		retryAfter := parseRetryAfterSeconds(resp.Header.Get("Retry-After"))
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("rate limited by Microsoft Graph: %s", string(body))

		if attempt == maxAttempts {
			return nil, lastErr
		}

		if retryAfter <= 0 {
			retryAfter = attempt
		}
		time.Sleep(time.Duration(retryAfter) * time.Second)
	}

	return nil, lastErr
}

func parseRetryAfterSeconds(value string) int {
	if value == "" {
		return 0
	}

	seconds, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}

	return seconds
}

func (od *OneDriveProvider) CreateFolder(remoteParentID string, name string) (physicalID string, err error) {
	var urlStr string
	if remoteParentID == "" || remoteParentID == "root" {
		urlStr = "https://graph.microsoft.com/v1.0/me/drive/root/children"
	} else {
		urlStr = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s/children", remoteParentID)
	}

	meta := map[string]interface{}{
		"name":   name,
		"folder": map[string]interface{}{},
		"@microsoft.graph.conflictBehavior": "rename",
	}

	bodyBytes, _ := json.Marshal(meta)
	resp, err := od.Client.Post(urlStr, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create folder failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

func (od *OneDriveProvider) UploadFile(remoteParentID string, filename string, r io.Reader, size int64) (physicalID string, err error) {
	// If the file is small (< 4MB), use a simple PUT upload.
	// Otherwise, we create an upload session to handle larger files chunk by chunk.
	const maxSimpleUpload = 4 * 1024 * 1024 // 4 MB

	if size <= maxSimpleUpload {
		var urlStr string
		if remoteParentID == "" || remoteParentID == "root" {
			urlStr = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/root:/%s:/content", url.PathEscape(filename))
		} else {
			urlStr = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s:/%s:/content", remoteParentID, url.PathEscape(filename))
		}

		req, err := http.NewRequest("PUT", urlStr, r)
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/octet-stream")

		resp, err := od.Client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("simple upload failed (%d): %s", resp.StatusCode, string(body))
		}

		var result struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", err
		}
		return result.ID, nil
	}

	// Large upload session path
	var sessionUrlStr string
	if remoteParentID == "" || remoteParentID == "root" {
		sessionUrlStr = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/root:/%s:/createUploadSession", url.PathEscape(filename))
	} else {
		sessionUrlStr = fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s:/%s:/createUploadSession", remoteParentID, url.PathEscape(filename))
	}

	sessionMeta := map[string]interface{}{
		"item": map[string]interface{}{
			"@microsoft.graph.conflictBehavior": "replace",
		},
	}
	sessionMetaBytes, _ := json.Marshal(sessionMeta)

	resp, err := od.Client.Post(sessionUrlStr, "application/json", bytes.NewBuffer(sessionMetaBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create upload session (%d): %s", resp.StatusCode, string(body))
	}

	var sessionResult struct {
		UploadUrl string `json:"uploadUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sessionResult); err != nil {
		return "", err
	}

	// Read & upload in 320 KB chunks (OneDrive chunks must be multiples of 320 KB / 327,680 bytes)
	const chunkSize = 327680 * 4 // ~1.3MB chunk size
	buf := make([]byte, chunkSize)
	var bytesUploaded int64 = 0

	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			chunkData := buf[:n]
			req, err := http.NewRequest("PUT", sessionResult.UploadUrl, bytes.NewReader(chunkData))
			if err != nil {
				return "", err
			}
			req.Header.Set("Content-Length", strconv.Itoa(n))
			req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", bytesUploaded, bytesUploaded+int64(n)-1, size))

			// Use the normal HTTP client (with graph auth headers) or direct client since session url already has authentication token
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
				body, _ := io.ReadAll(resp.Body)
				return "", fmt.Errorf("chunk upload failed (%d): %s", resp.StatusCode, string(body))
			}

			bytesUploaded += int64(n)

			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
				var finalResult struct {
					ID string `json:"id"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&finalResult); err == nil && finalResult.ID != "" {
					return finalResult.ID, nil
				}
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}

	return "", fmt.Errorf("upload session finished but did not return final item ID")
}

func (od *OneDriveProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	urlStr := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s/content", physicalID)
	resp, err := od.Client.Get(urlStr)
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

func (od *OneDriveProvider) DeleteFile(physicalID string) error {
	req, err := http.NewRequest("DELETE", "https://graph.microsoft.com/v1.0/me/drive/items/"+physicalID, nil)
	if err != nil {
		return err
	}

	resp, err := od.Client.Do(req)
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

func (od *OneDriveProvider) GetWebURL(physicalID string, isFolder bool) (string, error) {
	urlStr := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s/createLink", url.PathEscape(physicalID))
	reqBody := []byte(`{"type":"view","scope":"anonymous"}`)
	resp, err := od.Client.Post(urlStr, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create share link failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Link struct {
			WebURL string `json:"webUrl"`
		} `json:"link"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Link.WebURL, nil
}

func (od *OneDriveProvider) ListPermissions(physicalID string) ([]SharePermission, error) {
	urlStr := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s/permissions", url.PathEscape(physicalID))
	resp, err := od.Client.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list permissions failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Value []struct {
			ID    string   `json:"id"`
			Roles []string `json:"roles"`
			Link  *struct {
				Type  string `json:"type"`
				Scope string `json:"scope"`
			} `json:"link"`
			Invitation *struct {
				Email string `json:"email"`
			} `json:"invitation"`
			GrantedTo *struct {
				User *struct {
					Email       string `json:"email"`
					DisplayName string `json:"displayName"`
				} `json:"user"`
			} `json:"grantedTo"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var list []SharePermission
	for _, p := range result.Value {
		perm := SharePermission{
			ID: p.ID,
		}

		role := "reader"
		if len(p.Roles) > 0 {
			if p.Roles[0] == "write" {
				role = "writer"
			} else if p.Roles[0] == "owner" {
				role = "owner"
			}
		}
		perm.Role = role

		if p.Link != nil {
			perm.Type = "anyone"
			perm.DisplayName = fmt.Sprintf("Anyone with %s link", p.Link.Type)
		} else {
			perm.Type = "user"
			if p.Invitation != nil {
				perm.EmailAddress = p.Invitation.Email
			} else if p.GrantedTo != nil && p.GrantedTo.User != nil {
				perm.EmailAddress = p.GrantedTo.User.Email
				perm.DisplayName = p.GrantedTo.User.DisplayName
			}
		}

		list = append(list, perm)
	}
	return list, nil
}

func (od *OneDriveProvider) AddPermission(physicalID string, email string, role string) error {
	urlStr := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s/invite", url.PathEscape(physicalID))
	
	msRole := "read"
	if role == "writer" {
		msRole = "write"
	}

	reqBody := map[string]interface{}{
		"recipients": []map[string]string{
			{"email": email},
		},
		"roles": []string{msRole},
		"requireSignIn": true,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	resp, err := od.Client.Post(urlStr, "application/json", bytes.NewBuffer(bodyBytes))
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

func (od *OneDriveProvider) DeletePermission(physicalID string, permID string) error {
	urlStr := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s/permissions/%s", url.PathEscape(physicalID), url.PathEscape(permID))
	req, err := http.NewRequest("DELETE", urlStr, nil)
	if err != nil {
		return err
	}

	resp, err := od.Client.Do(req)
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

func (od *OneDriveProvider) SetGeneralAccess(physicalID string, accessType string, role string) error {
	if accessType == "anyone" {
		urlStr := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s/createLink", url.PathEscape(physicalID))
		
		odType := "view"
		if role == "writer" {
			odType = "edit"
		}

		reqBody := map[string]string{
			"type":  odType,
			"scope": "anonymous",
		}
		bodyBytes, _ := json.Marshal(reqBody)
		resp, err := od.Client.Post(urlStr, "application/json", bytes.NewBuffer(bodyBytes))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("set general access failed (%d): %s", resp.StatusCode, string(body))
		}
	} else {
		// Remove all anonymous links
		perms, err := od.ListPermissions(physicalID)
		if err != nil {
			return err
		}
		for _, p := range perms {
			if p.Type == "anyone" {
				_ = od.DeletePermission(physicalID, p.ID)
			}
		}
	}
	return nil
}

