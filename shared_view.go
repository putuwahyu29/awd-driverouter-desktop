package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"driverouter/backend/db"
	"driverouter/backend/provider"
	"driverouter/backend/sync"

	"github.com/google/uuid"
)

func (a *App) getSharedFiles(searchKeyword string) ([]db.FileRecord, error) {
	accounts, err := a.database.GetAccounts()
	if err != nil {
		return nil, err
	}

	var results []db.FileRecord
	seen := make(map[string]bool)
	searchLower := strings.ToLower(strings.TrimSpace(searchKeyword))

	for _, account := range accounts {
		if !account.Active {
			continue
		}

		client, err := sync.FetchActiveProviderClient(a.database, account, nil)
		if err != nil {
			continue
		}

		var sharedItems []provider.FileMetadata
		switch typed := client.(type) {
		case *provider.GoogleDriveProvider:
			sharedItems, err = listGoogleSharedFiles(typed)
		case *provider.OneDriveProvider:
			sharedItems, err = listOneDriveSharedFiles(typed)
		default:
			continue
		}
		if err != nil {
			continue
		}

		for _, item := range sharedItems {
			if searchLower != "" && !strings.Contains(strings.ToLower(item.Name), searchLower) {
				continue
			}

			stableID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(account.ID+":"+item.PhysicalID)).String()
			if seen[stableID] {
				continue
			}
			seen[stableID] = true

			results = append(results, db.FileRecord{
				ID:         stableID,
				Name:       item.Name,
				Size:       item.Size,
				IsFolder:   item.IsFolder,
				ParentID:   "shared",
				Provider:   account.Provider,
				AccountID:  account.ID,
				PhysicalID: item.PhysicalID,
				CreatedAt:  item.CreatedAt,
				ModifiedAt: item.ModifiedAt,
				Starred:    false,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if !results[i].ModifiedAt.Equal(results[j].ModifiedAt) {
			return results[i].ModifiedAt.After(results[j].ModifiedAt)
		}
		if results[i].Name != results[j].Name {
			return results[i].Name < results[j].Name
		}
		return results[i].ID < results[j].ID
	})

	return results, nil
}

func listGoogleSharedFiles(client *provider.GoogleDriveProvider) ([]provider.FileMetadata, error) {
	var results []provider.FileMetadata
	pageToken := ""

	for {
		query := "sharedWithMe = true and trashed = false"
		fields := "nextPageToken,files(id,name,size,mimeType,createdTime,modifiedTime)"
		urlStr := fmt.Sprintf("https://www.googleapis.com/drive/v3/files?q=%s&fields=%s&pageSize=1000", url.QueryEscape(query), url.QueryEscape(fields))
		if pageToken != "" {
			urlStr += "&pageToken=" + url.QueryEscape(pageToken)
		}

		resp, err := client.Client.Get(urlStr)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("google shared files request failed (%d): %s", resp.StatusCode, string(body))
		}

		var payload struct {
			NextPageToken string `json:"nextPageToken"`
			Files         []struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				Size         string `json:"size"`
				MimeType     string `json:"mimeType"`
				CreatedTime  string `json:"createdTime"`
				ModifiedTime string `json:"modifiedTime"`
			} `json:"files"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		for _, file := range payload.Files {
			created, _ := time.Parse(time.RFC3339, file.CreatedTime)
			modified, _ := time.Parse(time.RFC3339, file.ModifiedTime)
			size, _ := strconv.ParseInt(file.Size, 10, 64)
			results = append(results, provider.FileMetadata{
				PhysicalID: file.ID,
				Name:       file.Name,
				Size:       size,
				IsFolder:   file.MimeType == "application/vnd.google-apps.folder",
				Provider:   "google",
				CreatedAt:  created,
				ModifiedAt: modified,
			})
		}

		pageToken = payload.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return results, nil
}

func listOneDriveSharedFiles(client *provider.OneDriveProvider) ([]provider.FileMetadata, error) {
	resp, err := client.Client.Get("https://graph.microsoft.com/v1.0/me/drive/sharedWithMe")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("onedrive shared files request failed (%d): %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Value []struct {
			RemoteItem struct {
				ID                   string `json:"id"`
				Name                 string `json:"name"`
				Size                 int64  `json:"size"`
				Folder               *struct{} `json:"folder"`
				CreatedDateTime      string `json:"createdDateTime"`
				LastModifiedDateTime string `json:"lastModifiedDateTime"`
			} `json:"remoteItem"`
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	results := make([]provider.FileMetadata, 0, len(payload.Value))
	for _, item := range payload.Value {
		remote := item.RemoteItem
		if remote.ID == "" {
			remote.ID = item.ID
		}
		if remote.Name == "" {
			remote.Name = item.Name
		}
		created, _ := time.Parse(time.RFC3339, remote.CreatedDateTime)
		modified, _ := time.Parse(time.RFC3339, remote.LastModifiedDateTime)
		results = append(results, provider.FileMetadata{
			PhysicalID: remote.ID,
			Name:       remote.Name,
			Size:       remote.Size,
			IsFolder:   remote.Folder != nil,
			Provider:   "onedrive",
			CreatedAt:  created,
			ModifiedAt: modified,
		})
	}

	return results, nil
}
