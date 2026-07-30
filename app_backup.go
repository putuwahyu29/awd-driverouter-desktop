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
	"strings"
	gosync "sync"
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

func (a *App) RunSyncTaskNowByID(id string) error {
	task, err := a.database.GetSyncTaskByID(id)
	if err != nil {
		return fmt.Errorf("tugas backup tidak ditemukan: %w", err)
	}
	go a.RunSyncTaskNow(task)
	return nil
}

func isSystemOrIgnoredFile(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, "~$") || strings.HasPrefix(lower, ".tmp") || strings.HasSuffix(lower, ".tmp") {
		return true
	}
	switch lower {
	case "desktop.ini", "thumbs.db", ".ds_store", "icon\r", "ntuser.dat", "ntuser.dat.log1", "ntuser.dat.log2", "system volume information", "$recycle.bin", "my documents", "my videos", "my music", "my pictures", "application data", "local settings", "printhood", "nethood", "recent", "sendto", "start menu", "templates":
		return true
	}
	return false
}

func (a *App) resolveOrCreateRelativeFolderCached(baseFolderID string, relDir string, cache map[string]string) (string, error) {
	if relDir == "" || relDir == "." || relDir == "/" {
		return baseFolderID, nil
	}
	cleanRel := filepath.ToSlash(relDir)
	if id, ok := cache[cleanRel]; ok {
		return id, nil
	}
	parts := strings.Split(cleanRel, "/")
	currentParent := baseFolderID
	currentPath := ""
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		if cachedID, ok := cache[currentPath]; ok {
			currentParent = cachedID
			continue
		}

		existing, err := a.database.GetFileByNameAndParent(part, currentParent)
		if err == nil && existing.IsFolder {
			currentParent = existing.ID
			cache[currentPath] = existing.ID
		} else {
			newID, err := retryOn429(func() (string, error) {
				return a.CreateFolder(currentParent, part)
			})
			if err != nil {
				return "", fmt.Errorf("gagal membuat subfolder '%s': %w", part, err)
			}
			currentParent = newID
			cache[currentPath] = newID
		}
	}
	cache[cleanRel] = currentParent
	return currentParent, nil
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

	// Prevent backup loops on Virtual Drive letter
	if autoLetter, err := a.database.GetSetting("auto_mount_letter"); err == nil && autoLetter != "" {
		vol := filepath.VolumeName(task.LocalPath)
		cleanLetter := strings.TrimSuffix(strings.TrimSuffix(autoLetter, "\\"), "/")
		if strings.EqualFold(vol, cleanLetter) {
			log.Printf("Sync Task skipped: local path %s is inside Virtual Drive %s", task.LocalPath, autoLetter)
			_ = a.database.UpdateSyncTaskLastSync(task.ID, time.Now().Format("2006-01-02 15:04:05")+" (Virtual Drive diabaikan)")
			runtime.EventsEmit(a.ctx, "sync_tasks_updated", nil)
			return
		}
	}

	// Check local path existence
	if _, err := os.Stat(task.LocalPath); os.IsNotExist(err) {
		log.Printf("Sync Task local folder not found: %s", task.LocalPath)
		_ = a.database.UpdateSyncTaskLastSync(task.ID, time.Now().Format("2006-01-02 15:04:05")+" (Folder tidak ditemukan)")
		runtime.EventsEmit(a.ctx, "sync_tasks_updated", nil)
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
		newFolderID, err := retryOn429(func() (string, error) {
			return a.CreateFolder(task.TargetFolderID, folderName)
		})
		if err != nil {
			log.Printf("Sync Task failed to create parent folder '%s': %v", folderName, err)
			return
		}
		actualTargetFolderID = newFolderID
	}

	var skippedCount int
	localFilesMap := make(map[string]bool)
	folderCache := make(map[string]string)

	// Phase 1: Collect files to upload during walk (no network calls for uploads)
	type syncUploadJob struct {
		targetFolderID string
		path           string
		filename       string
		relPath        string
		accountID      string
	}
	var uploadJobs []syncUploadJob

	// A. Recursive Scan (Local -> Cloud) — collect phase
	walkErr := filepath.WalkDir(task.LocalPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			log.Printf("Sync Task permission/access warning at %s: %v", path, err)
			skippedCount++
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		filename := d.Name()
		if isSystemOrIgnoredFile(filename) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip NTFS Junction Points & Symlinks to prevent Access Denied on legacy Windows links
		if d.Type()&os.ModeSymlink != 0 || d.Type()&os.ModeIrregular != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(task.LocalPath, path)
		if err != nil {
			relPath = filename
		}
		relDir := filepath.Dir(relPath)
		localFilesMap[filepath.ToSlash(relPath)] = true

		targetFolderID, err := a.resolveOrCreateRelativeFolderCached(actualTargetFolderID, relDir, folderCache)
		if err != nil {
			log.Printf("Sync Task failed to resolve directory for %s: %v", relPath, err)
			skippedCount++
			return nil
		}

		// Check if file already exists in DB for target cloud folder
		existingFile, err := a.database.GetFileByNameAndParent(filename, targetFolderID)
		if err == nil && !existingFile.IsFolder && existingFile.ID != "" {
			// File already exists! Skip uploading to prevent duplicates.
			return nil
		}

		accID := task.AccountID
		if accID == "auto" {
			accID = ""
		}

		uploadJobs = append(uploadJobs, syncUploadJob{
			targetFolderID: targetFolderID,
			path:           path,
			filename:       filename,
			relPath:        relPath,
			accountID:      accID,
		})

		return nil
	})

	if walkErr != nil {
		log.Printf("Sync Task WalkDir notice: %v", walkErr)
	}

	// Phase 2: Upload collected files concurrently with bounded worker pool
	if len(uploadJobs) > 0 {
		log.Printf("Sync Task: uploading %d files with %d workers", len(uploadJobs), maxConcurrentUploads)
		sem := make(chan struct{}, maxConcurrentUploads)
		var wg gosync.WaitGroup

		for _, job := range uploadJobs {
			wg.Add(1)
			go func(j syncUploadJob) {
				defer wg.Done()
				sem <- struct{}{}        // acquire slot
				defer func() { <-sem }() // release slot

				log.Printf("Sync Task: Uploading %s to cloud", j.relPath)
				uploadID := uuid.New().String()
				runtime.EventsEmit(a.ctx, "upload_started", map[string]interface{}{"uploadId": uploadID, "filename": j.filename})

				upErr := a.uploadFileFromPath(j.targetFolderID, j.path, j.accountID, uploadID)
				if upErr != nil {
					log.Printf("Sync Task failed to upload %s: %v", j.relPath, upErr)
					runtime.EventsEmit(a.ctx, "upload_failed", map[string]string{"uploadId": uploadID, "filename": j.filename, "error": upErr.Error()})
					skippedCount++
				} else {
					log.Printf("Sync Task uploaded successfully: %s", j.relPath)
					runtime.EventsEmit(a.ctx, "upload_completed", map[string]interface{}{"uploadId": uploadID, "filename": j.filename})
				}
			}(job)
		}

		wg.Wait()
		log.Printf("Sync Task: finished uploading %d files", len(uploadJobs))
	}

	// B. Two-Way Sync Download (Cloud -> Local for top-level files)
	if task.SyncMode == "two-way" {
		virtualFiles, err := a.database.GetFiles(actualTargetFolderID, false, "")
		if err == nil {
			for _, fRec := range virtualFiles {
				if fRec.IsFolder {
					continue
				}
				if !localFilesMap[fRec.Name] {
					destPath := filepath.Join(task.LocalPath, fRec.Name)
					log.Printf("Sync Task (Two-Way): Downloading %s to local", fRec.Name)

					downloadID := uuid.New().String()
					runtime.EventsEmit(a.ctx, "download_started", map[string]interface{}{"downloadId": downloadID, "filename": fRec.Name})

					dlErr := a.downloadFileToLocalPath(fRec.ID, destPath)
					if dlErr != nil {
						log.Printf("Sync Task failed to download %s: %v", fRec.Name, dlErr)
						runtime.EventsEmit(a.ctx, "download_failed", map[string]string{"downloadId": downloadID, "filename": fRec.Name, "error": dlErr.Error()})
					} else {
						log.Printf("Sync Task downloaded successfully: %s", fRec.Name)
						runtime.EventsEmit(a.ctx, "download_completed", map[string]interface{}{"downloadId": downloadID, "filename": fRec.Name})
						_ = a.database.LogActivity("root", fRec.Name, "download", fmt.Sprintf("Backup otomatis: mengunduh '%s' dari cloud", fRec.Name))
					}
				}
			}
		}
	}

	nowStr := time.Now().Format("2006-01-02 15:04:05")
	if skippedCount > 0 {
		nowStr += fmt.Sprintf(" (%d berkas/folder diabaikan)", skippedCount)
	}
	_ = a.database.UpdateSyncTaskLastSync(task.ID, nowStr)
	runtime.EventsEmit(a.ctx, "sync_tasks_updated", nil)
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
