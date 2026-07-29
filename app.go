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
	ctx             context.Context
	database        *db.DB
	syncMgr         *sync.SyncManager
	uploadHub       *transferHub
	virtualDriveMgr *NativeVirtualDriveManager
	webServer       *WebServer
	quitting        bool
	minimizeToTray  bool
	isHeadless      bool
	headlessPort    int
	backupMu        gosync.Mutex
	backupTicker    *time.Ticker
	backupStop      chan struct{}
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
	a.virtualDriveMgr = NewNativeVirtualDriveManager(a)
	a.webServer = NewWebServer(a)
	if err := a.webServer.Start(); err != nil {
		log.Printf("Failed to start WebServer: %v", err)
	}

	// Load tray setting
	minToTrayStr, err := database.GetSetting("minimize_to_tray")
	if err == nil && minToTrayStr == "false" {
		a.minimizeToTray = false
	} else {
		a.minimizeToTray = true // default to true
	}

	// Ensure auto startup registry setting matches database preference
	autoStartupStr, err := database.GetSetting("auto_startup")
	if err == nil {
		if autoStartupStr == "true" && !a.IsStartupEnabled() {
			a.SetStartup(true)
		} else if autoStartupStr == "false" && a.IsStartupEnabled() {
			a.SetStartup(false)
		}
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

		// Auto-mount Virtual Drive on startup (default to true unless explicitly disabled)
		autoMountStr, err := database.GetSetting("auto_mount_drive")
		if err != nil || autoMountStr != "false" {
			autoLetter, _ := database.GetSetting("auto_mount_letter")
			if autoLetter == "" {
				autoLetter = "Z:"
			}
			log.Printf("Auto-mounting Virtual Drive %s on startup...", autoLetter)
			_, _ = a.MountVirtualDrive("all", autoLetter)
		}
	}()
}

func (a *App) shutdown(ctx context.Context) {
	// Cleanly unmount all virtual drives on app shutdown
	if a.virtualDriveMgr != nil {
		a.virtualDriveMgr.mu.Lock()
		for driveLetter := range a.virtualDriveMgr.mounts {
			log.Printf("Unmounting virtual drive %s on app shutdown...", driveLetter)
			_ = a.UnmountVirtualDrive(driveLetter)
		}
		a.virtualDriveMgr.mu.Unlock()
	}

	if a.webServer != nil {
		a.webServer.Stop()
	}
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

const AppVersion = "1.1.0"

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

// GetWebShares returns all active web share items.
func (a *App) GetWebShares() ([]WebShareItem, error) {
	if a.webServer == nil {
		return []WebShareItem{}, nil
	}
	a.webServer.Mu.Lock()
	defer a.webServer.Mu.Unlock()
	return a.webServer.SharedItems, nil
}

// CreateWebShare creates a new internal web share link for a file or folder.
func (a *App) CreateWebShare(virtualID string, password string) (WebShareItem, error) {
	if a.webServer == nil {
		return WebShareItem{}, fmt.Errorf("web server is not active")
	}

	f, err := a.database.GetFile(virtualID)
	if err != nil {
		return WebShareItem{}, fmt.Errorf("file not found: %w", err)
	}

	itemType := "file"
	if f.IsFolder {
		itemType = "folder"
	}

	item := WebShareItem{
		ID:          a.webServer.generateID(),
		Name:        f.Name,
		Type:        itemType,
		VirtualID:   f.ID,
		Size:        f.Size,
		Date:        time.Now().Unix(),
		Password:    password,
		AccessCount: 0,
	}

	a.webServer.Mu.Lock()
	a.webServer.SharedItems = append(a.webServer.SharedItems, item)
	a.webServer.Mu.Unlock()

	a.webServer.SaveShares()
	return item, nil
}

// DeleteWebShare removes a web share link by ID.
func (a *App) DeleteWebShare(shareID string) (bool, error) {
	if a.webServer == nil {
		return false, fmt.Errorf("web server is not active")
	}

	a.webServer.Mu.Lock()
	newItems := make([]WebShareItem, 0, len(a.webServer.SharedItems))
	found := false
	for _, item := range a.webServer.SharedItems {
		if item.ID == shareID {
			found = true
			continue
		}
		newItems = append(newItems, item)
	}
	a.webServer.SharedItems = newItems
	a.webServer.Mu.Unlock()

	if found {
		a.webServer.SaveShares()
	}
	return found, nil
}

// UpdateWebSharePassword updates the password protection of a web share.
func (a *App) UpdateWebSharePassword(shareID string, password string) error {
	if a.webServer == nil {
		return fmt.Errorf("web server is not active")
	}

	a.webServer.Mu.Lock()
	found := false
	for i := range a.webServer.SharedItems {
		if a.webServer.SharedItems[i].ID == shareID {
			a.webServer.SharedItems[i].Password = password
			found = true
			break
		}
	}
	a.webServer.Mu.Unlock()

	if !found {
		return fmt.Errorf("share item not found")
	}
	a.webServer.SaveShares()
	return nil
}

// TogglePublicTunnel starts or stops the Cloudflare public tunnel.
func (a *App) TogglePublicTunnel(enable bool) (string, error) {
	if a.webServer == nil {
		return "", fmt.Errorf("web server is not active")
	}

	if enable {
		return a.webServer.StartTunnel()
	} else {
		a.webServer.StopTunnel()
		return "", nil
	}
}

// GetTunnelPublicUrl returns the current public URL of the tunnel if running.
func (a *App) GetTunnelPublicUrl() string {
	if a.webServer == nil {
		return ""
	}
	a.webServer.Mu.Lock()
	defer a.webServer.Mu.Unlock()
	return a.webServer.PublicUrl
}

// IsTunnelRunning checks if the Cloudflare tunnel is active.
func (a *App) IsTunnelRunning() bool {
	if a.webServer == nil {
		return false
	}
	a.webServer.Mu.Lock()
	defer a.webServer.Mu.Unlock()
	return a.webServer.Tunneling
}

// GetLocalIPAddress returns the primary local network IP address.
func (a *App) GetLocalIPAddress() string {
	if a.webServer == nil {
		return "127.0.0.1"
	}
	return a.webServer.GetLocalIP()
}

// GetWebServerPort returns the HTTP server port.
func (a *App) GetWebServerPort() int {
	if a.webServer == nil {
		return 0
	}
	return a.webServer.Port
}


