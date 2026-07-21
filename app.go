package main

import (
	"context"
	"fmt"
	"log"
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

// GetWSAuthToken returns the WebSocket authentication token for the frontend.
func (a *App) GetWSAuthToken() string {
	if a.uploadHub == nil {
		return ""
	}
	return a.uploadHub.authToken
}
