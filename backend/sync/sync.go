package sync

import (
	"driverouter/backend/db"
	"driverouter/backend/provider"
	"driverouter/backend/router"
	"fmt"
	"log"
	gosync "sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var (
	tokenCacheMutex gosync.RWMutex
	tokenCache      = make(map[string]*oauth2.Token)
)

func GetCachedToken(accountID string) (*oauth2.Token, bool) {
	tokenCacheMutex.RLock()
	defer tokenCacheMutex.RUnlock()
	tok, found := tokenCache[accountID]
	return tok, found
}

func CacheToken(accountID string, tok *oauth2.Token) {
	tokenCacheMutex.Lock()
	defer tokenCacheMutex.Unlock()
	tokenCache[accountID] = tok
}

type SyncManager struct {
	Database *db.DB
}

func NewSyncManager(database *db.DB) *SyncManager {
	return &SyncManager{Database: database}
}

// SyncAccount crawls the cloud drive for the given account starting from the "driverouter" folder
// and updates the local SQLite database to represent the file hierarchy.
func (sm *SyncManager) SyncAccount(acc db.AccountRecord, p provider.Provider) error {
	log.Printf("Starting sync for account %s (%s)...", acc.DisplayName, acc.Email)

	driverouterFolderID := "root"

	// 2. Perform a recursive crawl of the AwdDriveRouter folder structure on this provider
	type queueItem struct {
		remoteParentID string
		virtualParentID string
		virtualPath    string // e.g. "", "/Documents"
	}

	queue := []queueItem{
		{remoteParentID: driverouterFolderID, virtualParentID: "root", virtualPath: ""},
	}

	// Keep track of visited physical items to detect cycles or issues
	visited := make(map[string]bool)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.remoteParentID] {
			continue
		}
		visited[current.remoteParentID] = true

		items, err := p.ListDirectory(current.remoteParentID)
		if err != nil {
			log.Printf("Warning: failed to list folder %s: %v", current.remoteParentID, err)
			continue
		}

		for _, item := range items {
			itemVirtualPath := current.virtualPath + "/" + item.Name

			// Check if this file already exists in our virtual directory structure.
			// To merge mirrored or existing files, we query by parent virtual ID and name.
			var existingFile db.FileRecord
			var found bool

			filesInParent, err := sm.Database.GetFiles(current.virtualParentID, false, "")
			if err == nil {
				for _, f := range filesInParent {
					if f.Name == item.Name && f.IsFolder == item.IsFolder {
						existingFile = f
						found = true
						break
					}
				}
			}

			virtualID := ""
			var physicalMap router.PhysicalIDsMap

			if found {
				virtualID = existingFile.ID
				physicalMap, _ = router.DeserializePhysicalIDs(existingFile.PhysicalID)
			} else {
				virtualID = uuid.New().String()
				physicalMap = make(router.PhysicalIDsMap)
			}

			// Add/Update this provider's mapping
			physicalMap[acc.ID] = item.PhysicalID
			serializedMap, _ := router.SerializePhysicalIDs(physicalMap)

			// Save/Update file record in local database
			record := db.FileRecord{
				ID:         virtualID,
				Name:       item.Name,
				Size:       item.Size,
				IsFolder:   item.IsFolder,
				ParentID:   current.virtualParentID,
				Provider:   item.Provider, // will set to merged or show provider icon
				AccountID:  acc.ID,
				PhysicalID: serializedMap,
				CreatedAt:  item.CreatedAt,
				ModifiedAt: item.ModifiedAt,
				Starred:    false,
				Shared:     item.Shared,
			}

			if found {
				record.Starred = existingFile.Starred
			}

			err = sm.Database.SaveFile(record)
			if err != nil {
				log.Printf("Failed to save file metadata to local db: %v", err)
				continue
			}

			// If it's a folder, enqueue it to crawl its children
			if item.IsFolder {
				queue = append(queue, queueItem{
					remoteParentID:  item.PhysicalID,
					virtualParentID: virtualID,
					virtualPath:     itemVirtualPath,
				})
			}
		}
	}

	log.Printf("Sync for account %s finished.", acc.DisplayName)
	return nil
}

// FetchActiveProviderClient turns an account record into an active provider client.
func FetchActiveProviderClient(database *db.DB, acc db.AccountRecord, onTokenRefresh func(*oauth2.Token)) (provider.Provider, error) {
	var tok oauth2.Token
	if cached, found := GetCachedToken(acc.ID); found {
		tok = *cached
	} else {
		if t, err := time.Parse(time.RFC3339, acc.TokenExpiry); err == nil {
			tok.Expiry = t
		}
		tok.AccessToken = acc.AccessToken
		tok.RefreshToken = acc.RefreshToken
		tok.TokenType = "Bearer"
	}

	// Fetch custom credentials from DB
	clientID := ""
	clientSecret := ""
	if database != nil {
		clientID, _ = database.GetSetting(acc.Provider + "_client_id")
		clientSecret, _ = database.GetSetting(acc.Provider + "_client_secret")
	}

	wrappedOnRefresh := func(newToken *oauth2.Token) {
		CacheToken(acc.ID, newToken)
		if onTokenRefresh != nil {
			onTokenRefresh(newToken)
		}
	}

	switch acc.Provider {
	case "google":
		return provider.NewGoogleDriveProvider(clientID, clientSecret, &tok, wrappedOnRefresh), nil
	case "onedrive":
		return provider.NewOneDriveProvider(clientID, clientSecret, &tok, wrappedOnRefresh), nil
	case "dropbox":
		return provider.NewDropboxProvider(clientID, clientSecret, &tok, wrappedOnRefresh), nil
	case "box":
		return provider.NewBoxProvider(clientID, clientSecret, &tok, wrappedOnRefresh), nil
	case "yandex":
		return provider.NewYandexProvider(clientID, clientSecret, &tok, wrappedOnRefresh), nil
	case "pcloud":
		return provider.NewPCloudProvider(clientID, clientSecret, &tok, wrappedOnRefresh), nil
	case "webdav":
		return provider.NewWebDAVProvider(acc.Email, acc.AccessToken, acc.RefreshToken), nil
	case "s3":
		return provider.NewS3Provider(acc.Email, acc.AccessToken, acc.RefreshToken, acc.TokenExpiry), nil
	case "telegram":
		return provider.NewTelegramProvider(acc.Email, acc.AccessToken), nil
	case "telegram_user":
		apiID := 0
		apiHash := ""
		if database != nil {
			apiIDStr, _ := database.GetSetting("telegram_api_id")
			apiHash, _ = database.GetSetting("telegram_api_hash")
			fmt.Sscanf(apiIDStr, "%d", &apiID)
		}
		onRefreshUser := func(newSession string) {
			acc.AccessToken = newSession
			if database != nil {
				_ = database.SaveAccount(acc)
			}
		}
		return provider.NewTelegramUserProvider(acc.Email, acc.AccessToken, apiID, apiHash, onRefreshUser), nil
	default:
		return nil, fmt.Errorf("unknown cloud provider: %s", acc.Provider)
	}
}

// SyncAllDrives triggers sync across all connected and active drives.
func (sm *SyncManager) SyncAllDrives() error {
	accounts, err := sm.Database.GetAccounts()
	if err != nil {
		return err
	}

	for _, acc := range accounts {
		if !acc.Active {
			continue
		}

		// Closure to save refreshed token in-memory only
		onRefresh := func(newToken *oauth2.Token) {
			CacheToken(acc.ID, newToken)
			log.Printf("Token refreshed and cached in-memory for account %s (%s)", acc.DisplayName, acc.Email)
		}

		p, err := FetchActiveProviderClient(sm.Database, acc, onRefresh)
		if err != nil {
			log.Printf("Error creating client for account %s: %v", acc.DisplayName, err)
			continue
		}

		// Update quota on each sync
		used, total, qErr := p.GetQuota()
		if qErr == nil {
			acc.UsedSpace = used
			acc.TotalSpace = total
			_ = sm.Database.SaveAccount(acc)
		}

		err = sm.SyncAccount(acc, p)
		if err != nil {
			log.Printf("Sync failed for account %s: %v", acc.DisplayName, err)
		}
	}
	return nil
}
