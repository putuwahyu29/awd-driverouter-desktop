package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	gosync "sync"
	"time"

	"driverouter/backend/db"
	"driverouter/backend/sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx            context.Context
	database       *db.DB
	syncMgr        *sync.SyncManager
	uploadHub      *transferHub
	quitting       bool
	minimizeToTray bool
	backupMu       gosync.Mutex
	backupTicker   *time.Ticker
	backupStop     chan struct{}
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	database, err := db.InitDB()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	a.database = database
	a.syncMgr = sync.NewSyncManager(database)

	// Load tray setting
	minToTrayStr, err := database.GetSetting("minimize_to_tray")
	if err == nil && minToTrayStr == "false" {
		a.minimizeToTray = false
	} else {
		a.minimizeToTray = true // default to true
	}

	a.startTray()
	a.startUploadWebSocketServer()
	a.StartBackupService()

	// Migrate plain-text tokens to encrypted storage (idempotent)
	go func() {
		database.MigrateEncryptTokens()
	}()

	// Trigger async sync on startup
	go func() {
		time.Sleep(2 * time.Second)
		_ = a.syncMgr.SyncAllDrives()
	}()
}

func (a *App) shutdown(ctx context.Context) {
	a.StopBackupService()
	a.stopUploadWebSocketServer()
	if a.database != nil {
		_ = a.database.Close()
	}
}

// BeforeClose intercepts close request to minimize to tray or request an in-app confirmation.
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	if a.quitting {
		return false
	}

	// Dynamic language loading
	lang, _ := a.database.GetSetting("language")
	if lang == "" {
		lang = "en"
	}

	// Reload tray setting in case it changed
	minToTrayStr, _ := a.database.GetSetting("minimize_to_tray")
	a.minimizeToTray = minToTrayStr == "true" || minToTrayStr == ""

	if a.minimizeToTray {
		runtime.WindowHide(ctx)
		return true // prevent window from actually closing
	}
	runtime.EventsEmit(ctx, "app:request-exit-confirm", map[string]string{"lang": lang})
	return true
}

// ToggleStarred toggles the starred setting.
func (a *App) ToggleStarred(virtualID string, starred bool) error {
	return a.database.UpdateStarred(virtualID, starred)
}

// GetRecentFiles retrieves recently modified files.
func (a *App) GetRecentFiles() ([]db.FileRecord, error) {
	return a.database.GetRecentFiles(30)
}

// SyncDrives triggers a full synchronization scan.
func (a *App) SyncDrives() error {
	return a.syncMgr.SyncAllDrives()
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

const AppVersion = "1.0.0"

type VersionCheckResult struct {
	HasUpdate     bool   `json:"has_update"`
	LatestVersion string `json:"latest_version"`
	UpdateURL     string `json:"update_url"`
	ReleaseNotes  string `json:"release_notes"`
}

func compareVersions(v1, v2 string) int {
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")
	for i := 0; i < len(parts1) || i < len(parts2); i++ {
		p1 := 0
		p2 := 0
		if i < len(parts1) {
			p1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			p2, _ = strconv.Atoi(parts2[i])
		}
		if p1 > p2 {
			return 1
		}
		if p1 < p2 {
			return -1
		}
	}
	return 0
}

func (a *App) CheckForUpdates() (VersionCheckResult, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/putuwahyu29/awd-driverouter-desktop/releases/latest", nil)
	if err != nil {
		return VersionCheckResult{}, err
	}
	req.Header.Set("User-Agent", "awd-driverouter-desktop-updater")

	resp, err := client.Do(req)
	if err != nil {
		return VersionCheckResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return VersionCheckResult{}, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Body    string `json:"body"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return VersionCheckResult{}, err
	}

	hasUpdate := compareVersions(release.TagName, AppVersion) > 0

	return VersionCheckResult{
		HasUpdate:     hasUpdate,
		LatestVersion: release.TagName,
		UpdateURL:     release.HTMLURL,
		ReleaseNotes:  release.Body,
	}, nil
}

func (a *App) OpenReleaseURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

func (a *App) GetAppVersion() string {
	return AppVersion
}

// GetWSAuthToken returns the WebSocket authentication token for the frontend.
func (a *App) GetWSAuthToken() string {
	if a.uploadHub == nil {
		return ""
	}
	return a.uploadHub.authToken
}

