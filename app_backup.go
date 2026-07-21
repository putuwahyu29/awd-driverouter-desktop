package main

import (
	"driverouter/backend/db"
	"driverouter/backend/router"
	"driverouter/backend/sync"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) SelectBackupFolder() string {
	res, _ := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Folder for Backup",
	})
	return res
}

func (a *App) GetSyncTasks() ([]db.SyncTask, error) {
	return a.database.GetSyncTasks()
}

func (a *App) AddSyncTask(localPath, targetFolderID, accountID, syncMode string) error {
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("folder lokal tidak ditemukan")
	}

	task := db.SyncTask{
		ID:             uuid.New().String(),
		LocalPath:      localPath,
		TargetFolderID: targetFolderID,
		AccountID:      accountID,
		SyncMode:       syncMode,
		Enabled:        true,
		LastSync:       "",
	}

	err := a.database.AddSyncTask(task)
	if err != nil {
		return err
	}

	// Trigger immediate check in background
	go a.RunSyncTaskNow(task)
	return nil
}

func (a *App) RemoveSyncTask(id string) error {
	return a.database.DeleteSyncTask(id)
}

func (a *App) UpdateSyncTask(id string, targetFolderID string, accountID string, syncMode string) error {
	return a.database.UpdateSyncTask(id, targetFolderID, accountID, syncMode)
}

func (a *App) ToggleSyncTask(id string, enabled bool) error {
	return a.database.ToggleSyncTask(id, enabled)
}

func (a *App) GetBackupInterval() int {
	val, err := a.database.GetSetting("backup_interval")
	if err != nil || val == "" {
		return 60
	}
	var interval int
	if _, err := fmt.Sscan(val, &interval); err == nil && interval >= 5 {
		return interval
	}
	return 60
}

func (a *App) UpdateBackupInterval(seconds int) error {
	err := a.database.SaveSetting("backup_interval", fmt.Sprintf("%d", seconds))
	if err != nil {
		return err
	}
	a.StartBackupService()
	return nil
}

func (a *App) StartBackupService() {
	a.backupMu.Lock()
	defer a.backupMu.Unlock()

	// Stop existing service if running (without holding the lock for nested call)
	if a.backupStop != nil {
		close(a.backupStop)
		a.backupStop = nil
	}
	if a.backupTicker != nil {
		a.backupTicker.Stop()
		a.backupTicker = nil
	}

	intervalSeconds := a.GetBackupInterval()
	log.Printf("Starting backup service with interval of %d seconds...", intervalSeconds)

	a.backupTicker = time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	a.backupStop = make(chan struct{})

	ticker := a.backupTicker
	stop := a.backupStop

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Backup service goroutine panicked and recovered: %v", r)
			}
		}()

		// Run once on startup
		time.Sleep(5 * time.Second)
		a.runAllBackupTasks()

		for {
			select {
			case <-ticker.C:
				a.runAllBackupTasks()
			case <-stop:
				return
			case <-a.ctx.Done():
				return
			}
		}
	}()
	log.Println("Backup service started.")
}

func (a *App) StopBackupService() {
	a.backupMu.Lock()
	defer a.backupMu.Unlock()

	if a.backupStop != nil {
		close(a.backupStop)
		a.backupStop = nil
	}
	if a.backupTicker != nil {
		a.backupTicker.Stop()
		a.backupTicker = nil
	}
	log.Println("Backup service stopped.")
}

func (a *App) runAllBackupTasks() {
	tasks, err := a.database.GetSyncTasks()
	if err != nil {
		log.Printf("Failed to get sync tasks: %v", err)
		return
	}

	// Perform a single SyncAllDrives before processing all tasks (was previously called per-task)
	if len(tasks) > 0 {
		if err := a.syncMgr.SyncAllDrives(); err != nil {
			log.Printf("Backup pre-sync failed: %v", err)
		}
	}

	for _, task := range tasks {
		if !task.Enabled {
			continue
		}
		a.runSyncTaskOnly(task)
	}
}

// runSyncTaskOnly runs a single sync task WITHOUT triggering SyncAllDrives (deduplication).
func (a *App) RunSyncTaskNow(task db.SyncTask) {
	log.Printf("Executing Sync Task: %s (%s)", task.LocalPath, task.SyncMode)

	// Sync metadata from all cloud providers first to discover files added by other devices
	if err := a.syncMgr.SyncAllDrives(); err != nil {
		log.Printf("Sync Task metadata sync failed: %v", err)
	}

	a.runSyncTaskOnly(task)
}

// runSyncTaskOnly executes the actual file sync logic without re-triggering SyncAllDrives.
func (a *App) runSyncTaskOnly(task db.SyncTask) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Sync task goroutine panicked for %s: %v", task.LocalPath, r)
		}
	}()

	log.Printf("Running sync task: %s (%s)", task.LocalPath, task.SyncMode)

	// Check local path
	if _, err := os.Stat(task.LocalPath); os.IsNotExist(err) {
		log.Printf("Sync Task local folder not found: %s", task.LocalPath)
		return
	}

	// Resolve/create folder named after the local folder on target destination
	folderName := filepath.Base(task.LocalPath)
	if folderName == "" || folderName == "." || folderName == "/" {
		folderName = "Backup"
	}

	actualTargetFolderID := task.TargetFolderID
	existingFolder, err := a.database.GetFileByNameAndParent(folderName, task.TargetFolderID)
	if err == nil && existingFolder.IsFolder {
		actualTargetFolderID = existingFolder.ID
	} else {
		newFolderID, err := a.CreateFolder(task.TargetFolderID, folderName)
		if err != nil {
			log.Printf("Sync Task failed to create parent folder '%s': %v", folderName, err)
			return
		}
		actualTargetFolderID = newFolderID
	}

	// 1. Get virtual files in the target folder from DB
	virtualFiles, err := a.database.GetFiles(actualTargetFolderID, false, "")
	if err != nil {
		log.Printf("Sync Task failed to read target folder files: %v", err)
		return
	}

	virtualFilesMap := make(map[string]db.FileRecord)
	for _, f := range virtualFiles {
		if !f.IsFolder {
			virtualFilesMap[f.Name] = f
		}
	}

	// 2. Scan local files
	localFiles, err := os.ReadDir(task.LocalPath)
	if err != nil {
		log.Printf("Sync Task failed to read local folder: %v", err)
		return
	}

	localFilesMap := make(map[string]bool)

	// A. One-Way Upload (Local -> Cloud)
	for _, entry := range localFiles {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		localFilesMap[name] = true

		if _, exists := virtualFilesMap[name]; !exists {
			// File does not exist in Cloud: Upload it!
			fullPath := filepath.Join(task.LocalPath, name)
			log.Printf("Sync Task: Uploading %s to cloud", name)

			uploadID := uuid.New().String()
			runtime.EventsEmit(a.ctx, "upload_started", map[string]interface{}{"uploadId": uploadID, "filename": name})

			accID := task.AccountID
			if accID == "auto" {
				accID = "" // Let router auto allocate
			}

			err := a.uploadFileFromPath(actualTargetFolderID, fullPath, accID, uploadID)
			if err != nil {
				log.Printf("Sync Task failed to upload %s: %v", name, err)
				runtime.EventsEmit(a.ctx, "upload_failed", map[string]string{"uploadId": uploadID, "filename": name, "error": err.Error()})
			} else {
				log.Printf("Sync Task uploaded successfully: %s", name)
				runtime.EventsEmit(a.ctx, "upload_completed", map[string]interface{}{"uploadId": uploadID, "filename": name})
			}
		}
	}

	// B. Two-Way Download (Cloud -> Local)
	if task.SyncMode == "two-way" {
		for name, fRec := range virtualFilesMap {
			if !localFilesMap[name] {
				// File exists in Cloud but not locally: Download it!
				destPath := filepath.Join(task.LocalPath, name)
				log.Printf("Sync Task (Two-Way): Downloading %s to local", name)

				downloadID := uuid.New().String()
				runtime.EventsEmit(a.ctx, "download_started", map[string]interface{}{"downloadId": downloadID, "filename": name})

				err := a.downloadFileToLocalPath(fRec.ID, destPath)
				if err != nil {
					log.Printf("Sync Task failed to download %s: %v", name, err)
					runtime.EventsEmit(a.ctx, "download_failed", map[string]string{"downloadId": downloadID, "filename": name, "error": err.Error()})
				} else {
					log.Printf("Sync Task downloaded successfully: %s", name)
					runtime.EventsEmit(a.ctx, "download_completed", map[string]interface{}{"downloadId": downloadID, "filename": name})
					_ = a.database.LogActivity("root", name, "download", fmt.Sprintf("Backup otomatis: mengunduh '%s' dari penyimpanan cloud", name))
				}
			}
		}
	}

	// Update last sync time
	nowStr := time.Now().Format("2006-01-02 15:04:05")
	_ = a.database.UpdateSyncTaskLastSync(task.ID, nowStr)
}

func (a *App) downloadFileToLocalPath(virtualID string, destPath string) error {
	f, err := a.database.GetFile(virtualID)
	if err != nil {
		return err
	}

	physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
	if err != nil || len(physicalMap) == 0 {
		return fmt.Errorf("no physical file links found")
	}

	// Pick first available active account
	var activeAccID string
	var physID string
	var activeAcc db.AccountRecord

	accounts, _ := a.database.GetAccounts()
	for accID, pID := range physicalMap {
		for _, acc := range accounts {
			if acc.ID == accID && acc.Active {
				activeAccID = accID
				physID = pID
				activeAcc = acc
				break
			}
		}
		if activeAccID != "" {
			break
		}
	}

	if activeAccID == "" {
		return fmt.Errorf("no active accounts hold a copy")
	}

	p, err := sync.FetchActiveProviderClient(a.database, activeAcc, nil)
	if err != nil {
		return err
	}

	reader, err := p.DownloadFile(physID)
	if err != nil {
		return err
	}
	defer reader.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, reader)
	return err
}
