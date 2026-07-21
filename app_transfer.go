package main

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	gosync "sync"
	"strings"
	"time"

	"driverouter/backend/db"
	"driverouter/backend/router"
	"driverouter/backend/sync"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// UploadFileDialog triggers OS file picker and uploads selected file.
func (a *App) UploadFileDialog(parentID string, manualAccountID string) error {
	if parentID == "" {
		parentID = "root"
	}

	selectedFile, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select File to Upload to Awd DriveRouter",
	})
	if err != nil || selectedFile == "" {
		return err
	}

	filename := filepath.Base(selectedFile)
	uploadID := uuid.New().String()
	runtime.EventsEmit(a.ctx, "upload_started", map[string]interface{}{"uploadId": uploadID, "filename": filename})

	err = a.uploadFileFromPath(parentID, selectedFile, manualAccountID, uploadID)
	if err != nil {
		runtime.EventsEmit(a.ctx, "upload_failed", map[string]string{
			"uploadId": uploadID,
			"filename": filename,
			"error":    err.Error(),
		})
		return err
	}

	runtime.EventsEmit(a.ctx, "upload_completed", map[string]interface{}{"uploadId": uploadID, "filename": filename})
	return nil
}

// UploadFileFromPath allows the frontend (like drag-and-drop) to upload a file directly from a local path.
func (a *App) UploadFileFromPath(parentID string, filePath string) error {
	if parentID == "" {
		parentID = "root"
	}

	filename := filepath.Base(filePath)
	uploadID := uuid.New().String()
	runtime.EventsEmit(a.ctx, "upload_started", map[string]interface{}{"uploadId": uploadID, "filename": filename})

	err := a.uploadFileFromPath(parentID, filePath, "", uploadID)
	if err != nil {
		runtime.EventsEmit(a.ctx, "upload_failed", map[string]string{
			"uploadId": uploadID,
			"filename": filename,
			"error":    err.Error(),
		})
		return err
	}

	runtime.EventsEmit(a.ctx, "upload_completed", map[string]interface{}{"uploadId": uploadID, "filename": filename})
	return nil
}

// DownloadFileDialog downloads a virtual file to the selected local path.
func (a *App) DownloadFileDialog(virtualID string) error {
	f, err := a.database.GetFile(virtualID)
	if err != nil {
		return err
	}

	if f.IsFolder {
		return fmt.Errorf("directories cannot be downloaded directly. download individual files instead")
	}

	physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
	if err != nil || len(physicalMap) == 0 {
		return fmt.Errorf("no physical file links found for this record")
	}

	// Pick first available active account in map
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
		return fmt.Errorf("no active accounts hold a copy of this file")
	}

	// Trigger Save File dialog
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Download File",
		DefaultFilename: f.Name,
	})
	if err != nil || savePath == "" {
		return err
	}

	// Start downloading
	p, err := sync.FetchActiveProviderClient(a.database, activeAcc, nil)
	if err != nil {
		return err
	}

	downloadID := fmt.Sprintf("dl-%d", time.Now().UnixNano())
	runtime.EventsEmit(a.ctx, "download_started", map[string]string{
		"downloadId": downloadID,
		"filename":   f.Name,
	})

	runtime.LogInfof(a.ctx, "Downloading %s from %s...", f.Name, activeAcc.DisplayName)
	reader, err := p.DownloadFile(physID)
	if err != nil {
		if a.uploadHub != nil {
			a.uploadHub.broadcastProgress(progressMessage{
				Type:     "download_progress",
				ID:       downloadID,
				Filename: f.Name,
				Percent:  0,
				Error:    err.Error(),
			})
		}
		runtime.EventsEmit(a.ctx, "download_failed", map[string]string{
			"downloadId": downloadID,
			"filename":   f.Name,
			"error":      err.Error(),
		})
		return fmt.Errorf("failed to download from provider: %w", err)
	}
	defer reader.Close()

	out, err := os.Create(savePath)
	if err != nil {
		if a.uploadHub != nil {
			a.uploadHub.broadcastProgress(progressMessage{
				Type:     "download_progress",
				ID:       downloadID,
				Filename: f.Name,
				Percent:  0,
				Error:    err.Error(),
			})
		}
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer out.Close()

	var tracker *progressTracker
	var trackedReader io.Reader = reader
	if a.uploadHub != nil {
		tracker = newProgressTracker(a.uploadHub, downloadID, f.Name, f.Size, "download_progress")
		trackedReader = tracker.reader(reader)
	}

	_, err = io.Copy(out, trackedReader)
	if err != nil {
		if a.uploadHub != nil {
			a.uploadHub.broadcastProgress(progressMessage{
				Type:     "download_progress",
				ID:       downloadID,
				Filename: f.Name,
				Percent:  0,
				Error:    err.Error(),
			})
		}
		runtime.EventsEmit(a.ctx, "download_failed", map[string]string{
			"downloadId": downloadID,
			"filename":   f.Name,
			"error":      err.Error(),
		})
		return fmt.Errorf("failed to write file to disk: %w", err)
	}

	if a.uploadHub != nil {
		a.uploadHub.broadcastProgress(progressMessage{
			Type:     "download_progress",
			ID:       downloadID,
			Filename: f.Name,
			Percent:  100,
		})
	}

	runtime.EventsEmit(a.ctx, "download_completed", map[string]string{
		"downloadId": downloadID,
		"filename":   f.Name,
	})

	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "Download Complete",
		Message: fmt.Sprintf("Successfully downloaded '%s' to %s", f.Name, savePath),
	})

	_ = a.database.LogActivity(f.ID, f.Name, "download", fmt.Sprintf("Downloaded to local path: %s", savePath))
	return nil
}

// DownloadBulkDialog opens a folder picker and downloads multiple files to it.
func (a *App) DownloadBulkDialog(virtualIDs []string) error {
	if len(virtualIDs) == 0 {
		return nil
	}

	targetDir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Folder to Save Downloads",
	})
	if err != nil || targetDir == "" {
		return err
	}

	go func() {
		successCount := 0
		var failedFiles []string

		for _, id := range virtualIDs {
			f, err := a.database.GetFile(id)
			if err != nil {
				failedFiles = append(failedFiles, fmt.Sprintf("ID %s: %v", id, err))
				continue
			}

			if f.IsFolder {
				failedFiles = append(failedFiles, fmt.Sprintf("%s (Folder: skipped)", f.Name))
				continue
			}

			physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
			if err != nil || len(physicalMap) == 0 {
				failedFiles = append(failedFiles, fmt.Sprintf("%s (No copies found)", f.Name))
				continue
			}

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
				failedFiles = append(failedFiles, fmt.Sprintf("%s (No active provider account)", f.Name))
				continue
			}

			savePath := filepath.Join(targetDir, f.Name)
			downloadID := fmt.Sprintf("dl-%d", time.Now().UnixNano())

			runtime.EventsEmit(a.ctx, "download_started", map[string]string{
				"downloadId": downloadID,
				"filename":   f.Name,
			})

			p, err := sync.FetchActiveProviderClient(a.database, activeAcc, nil)
			if err != nil {
				if a.uploadHub != nil {
					a.uploadHub.broadcastProgress(progressMessage{
						Type:     "download_progress",
						ID:       downloadID,
						Filename: f.Name,
						Percent:  0,
						Error:    err.Error(),
					})
				}
				runtime.EventsEmit(a.ctx, "download_failed", map[string]string{
					"downloadId": downloadID,
					"filename":   f.Name,
					"error":      err.Error(),
				})
				failedFiles = append(failedFiles, fmt.Sprintf("%s (%v)", f.Name, err))
				continue
			}

			reader, err := p.DownloadFile(physID)
			if err != nil {
				if a.uploadHub != nil {
					a.uploadHub.broadcastProgress(progressMessage{
						Type:     "download_progress",
						ID:       downloadID,
						Filename: f.Name,
						Percent:  0,
						Error:    err.Error(),
					})
				}
				runtime.EventsEmit(a.ctx, "download_failed", map[string]string{
					"downloadId": downloadID,
					"filename":   f.Name,
					"error":      err.Error(),
				})
				failedFiles = append(failedFiles, fmt.Sprintf("%s (%v)", f.Name, err))
				continue
			}

			out, err := os.Create(savePath)
			if err != nil {
				reader.Close()
				if a.uploadHub != nil {
					a.uploadHub.broadcastProgress(progressMessage{
						Type:     "download_progress",
						ID:       downloadID,
						Filename: f.Name,
						Percent:  0,
						Error:    err.Error(),
					})
				}
				runtime.EventsEmit(a.ctx, "download_failed", map[string]string{
					"downloadId": downloadID,
					"filename":   f.Name,
					"error":      err.Error(),
				})
				failedFiles = append(failedFiles, fmt.Sprintf("%s (%v)", f.Name, err))
				continue
			}

			var tracker *progressTracker
			var trackedReader io.Reader = reader
			if a.uploadHub != nil {
				tracker = newProgressTracker(a.uploadHub, downloadID, f.Name, f.Size, "download_progress")
				trackedReader = tracker.reader(reader)
			}

			_, err = io.Copy(out, trackedReader)
			out.Close()
			reader.Close()

			if err != nil {
				if a.uploadHub != nil {
					a.uploadHub.broadcastProgress(progressMessage{
						Type:     "download_progress",
						ID:       downloadID,
						Filename: f.Name,
						Percent:  0,
						Error:    err.Error(),
					})
				}
				runtime.EventsEmit(a.ctx, "download_failed", map[string]string{
					"downloadId": downloadID,
					"filename":   f.Name,
					"error":      err.Error(),
				})
				failedFiles = append(failedFiles, fmt.Sprintf("%s (%v)", f.Name, err))
				continue
			}

			if a.uploadHub != nil {
				a.uploadHub.broadcastProgress(progressMessage{
					Type:     "download_progress",
					ID:       downloadID,
					Filename: f.Name,
					Percent:  100,
				})
			}

			runtime.EventsEmit(a.ctx, "download_completed", map[string]string{
				"downloadId": downloadID,
				"filename":   f.Name,
			})

			successCount++
		}

		var msg string
		if len(failedFiles) > 0 {
			msg = fmt.Sprintf("Successfully downloaded %d file(s).\n\nFailed / skipped %d item(s):\n- %s", successCount, len(failedFiles), strings.Join(failedFiles, "\n- "))
		} else {
			msg = fmt.Sprintf("Successfully downloaded all %d file(s) to:\n%s", successCount, targetDir)
		}

		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   "Bulk Download Complete",
			Message: msg,
		})
	}()

	return nil
}

func (a *App) uploadFileFromPath(parentVirtualID string, localPath string, manualAccountID string, uploadID string) error {
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	filename := filepath.Base(localPath)
	size := fileInfo.Size()

	// Select target account(s) using strategy
	targets, strategyName, err := router.SelectTargetAccounts(a.database, manualAccountID)
	if err != nil {
		return err
	}
	progressTracker := newProgressTracker(a.uploadHub, uploadID, filename, size*int64(len(targets)), "upload_progress")

	// Resolve the physical parent ID for each target account
	parentPhysicalIDs := make(map[string]string) // accountID -> physicalParentID
	for _, target := range targets {
		physParentID, err := a.resolvePhysicalFolder(parentVirtualID, target)
		if err != nil {
			return fmt.Errorf("failed to resolve directory tree on %s: %w", target.DisplayName, err)
		}
		parentPhysicalIDs[target.ID] = physParentID
	}

	// Upload to each target account concurrently
	uploadedPhysicalIDs := make(router.PhysicalIDsMap)
	var uploadMu gosync.Mutex
	var wg gosync.WaitGroup
	var firstErr error

	for _, target := range targets {
		wg.Add(1)
		go func(target db.AccountRecord) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					uploadMu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("panic during upload to %s: %v", target.DisplayName, r)
					}
					uploadMu.Unlock()
				}
			}()

			// Open file handle for each upload
			file, err := os.Open(localPath)
			if err != nil {
				uploadMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("failed to open source file: %w", err)
				}
				uploadMu.Unlock()
				return
			}

			p, err := sync.FetchActiveProviderClient(a.database, target, nil)
			if err != nil {
				file.Close()
				uploadMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				uploadMu.Unlock()
				return
			}

			physParentID := parentPhysicalIDs[target.ID]
			runtime.LogInfof(a.ctx, "Uploading %s to %s (%s)...", filename, target.DisplayName, strategyName)

			physID, err := p.UploadFile(physParentID, filename, progressTracker.reader(file), size)
			file.Close()
			if err != nil {
				if a.uploadHub != nil {
					a.uploadHub.broadcastProgress(progressMessage{
						Type:     "upload_progress",
						ID:       uploadID,
						Filename: filename,
						Percent:  0,
						Error:    err.Error(),
					})
				}
				uploadMu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("failed upload to %s: %w", target.DisplayName, err)
				}
				uploadMu.Unlock()
				return
			}

			uploadMu.Lock()
			uploadedPhysicalIDs[target.ID] = physID
			uploadMu.Unlock()
		}(target)
	}

	wg.Wait()

	if firstErr != nil {
		return firstErr
	}

	if a.uploadHub != nil {
		a.uploadHub.broadcastProgress(progressMessage{
			Type:     "upload_progress",
			ID:       uploadID,
			Filename: filename,
			Percent:  100,
		})
	}

	serializedPhysIDs, _ := router.SerializePhysicalIDs(uploadedPhysicalIDs)

	// Save record to local virtual database
	newFileRecord := db.FileRecord{
		ID:         uuid.New().String(),
		Name:       filename,
		Size:       size,
		IsFolder:   false,
		ParentID:   parentVirtualID,
		Provider:   targets[0].Provider, // primary provider representation
		AccountID:  targets[0].ID,
		PhysicalID: serializedPhysIDs,
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
	}

	if len(targets) > 1 {
		newFileRecord.Provider = "merged"
	}

	err = a.database.SaveFile(newFileRecord)
	if err != nil {
		return fmt.Errorf("failed to record file metadata: %w", err)
	}

	var targetNames string
	if len(targets) > 1 {
		names := make([]string, len(targets))
		for i, t := range targets {
			names[i] = fmt.Sprintf("%s (%s)", t.DisplayName, t.Provider)
		}
		targetNames = " (mirroring ke: " + strings.Join(names, ", ") + ")"
	} else if len(targets) == 1 {
		targetNames = fmt.Sprintf(" ke %s (%s)", targets[0].DisplayName, targets[0].Provider)
	}

	_ = a.database.LogActivity(newFileRecord.ID, newFileRecord.Name, "upload", fmt.Sprintf("Mengunggah berkas (%s)%s", formatBytes(newFileRecord.Size), targetNames))

	return nil
}

// resolvePhysicalFolder finds or creates the matching physical path sequence on the target provider.
func (a *App) resolvePhysicalFolder(virtualFolderID string, account db.AccountRecord) (string, error) {
	if virtualFolderID == "root" || virtualFolderID == "" {
		return "root", nil
	}

	// Recursive lookup of directory chain
	vFolder, err := a.database.GetFile(virtualFolderID)
	if err != nil {
		return "", err
	}

	physicalMap, err := router.DeserializePhysicalIDs(vFolder.PhysicalID)
	if err == nil {
		if physID, exists := physicalMap[account.ID]; exists {
			return physID, nil
		}
	}

	// Folder does not exist physically on this account yet.
	// Resolve its parent folder first.
	physParentID, err := a.resolvePhysicalFolder(vFolder.ParentID, account)
	if err != nil {
		return "", err
	}

	// Create folder physically on provider
	p, err := sync.FetchActiveProviderClient(a.database, account, nil)
	if err != nil {
		return "", err
	}

	physID, err := p.CreateFolder(physParentID, vFolder.Name)
	if err != nil {
		return "", err
	}

	// Save physical mapping back to DB
	if physicalMap == nil {
		physicalMap = make(router.PhysicalIDsMap)
	}
	physicalMap[account.ID] = physID
	vFolder.PhysicalID, _ = router.SerializePhysicalIDs(physicalMap)
	_ = a.database.SaveFile(vFolder)

	return physID, nil
}

// CopyFileToAccount streams a file from its source cloud account directly to the target cloud account.
func (a *App) CopyFileToAccount(virtualFileID string, destAccountID string, destParentVirtualID string) error {
	f, err := a.database.GetFile(virtualFileID)
	if err != nil {
		return err
	}

	if f.IsFolder {
		return fmt.Errorf("directories cannot be copied directly yet")
	}

	physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
	if err != nil || len(physicalMap) == 0 {
		return fmt.Errorf("no physical file links found for this record")
	}

	var srcAccID string
	var srcPhysID string
	var srcAcc db.AccountRecord

	accounts, _ := a.database.GetAccounts()
	for accID, pID := range physicalMap {
		for _, acc := range accounts {
			if acc.ID == accID && acc.Active {
				srcAccID = accID
				srcPhysID = pID
				srcAcc = acc
				break
			}
		}
		if srcAccID != "" {
			break
		}
	}

	if srcAccID == "" {
		return fmt.Errorf("no active provider account holds a copy of this file")
	}

	var destAcc db.AccountRecord
	foundDest := false
	for _, acc := range accounts {
		if acc.ID == destAccountID {
			destAcc = acc
			foundDest = true
			break
		}
	}

	if !foundDest {
		return fmt.Errorf("destination account not found")
	}

	if !destAcc.Active {
		return fmt.Errorf("destination account is inactive")
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("CopyFileToAccount goroutine panicked: %v", r)
			}
		}()
		transferID := fmt.Sprintf("tf-%d", time.Now().UnixNano())
		runtime.EventsEmit(a.ctx, "transfer_started", map[string]string{
			"transferId": transferID,
			"filename":   f.Name,
		})

		pSrc, err := sync.FetchActiveProviderClient(a.database, srcAcc, nil)
		if err != nil {
			a.sendTransferError(transferID, f.Name, err)
			return
		}

		reader, err := pSrc.DownloadFile(srcPhysID)
		if err != nil {
			a.sendTransferError(transferID, f.Name, err)
			return
		}
		defer reader.Close()

		destPhysParentID, err := a.resolvePhysicalFolder(destParentVirtualID, destAcc)
		if err != nil {
			a.sendTransferError(transferID, f.Name, err)
			return
		}

		pDest, err := sync.FetchActiveProviderClient(a.database, destAcc, nil)
		if err != nil {
			a.sendTransferError(transferID, f.Name, err)
			return
		}

		var tracker *progressTracker
		var trackedReader io.Reader = reader
		if a.uploadHub != nil {
			tracker = newProgressTracker(a.uploadHub, transferID, f.Name, f.Size, "transfer_progress")
			trackedReader = tracker.reader(reader)
		}

		destPhysID, err := pDest.UploadFile(destPhysParentID, f.Name, trackedReader, f.Size)
		if err != nil {
			a.sendTransferError(transferID, f.Name, err)
			return
		}

		destPhysicalMap := make(router.PhysicalIDsMap)
		destPhysicalMap[destAcc.ID] = destPhysID
		serializedDestPhysIDs, _ := router.SerializePhysicalIDs(destPhysicalMap)

		newFileRecord := db.FileRecord{
			ID:         uuid.New().String(),
			Name:       f.Name,
			Size:       f.Size,
			IsFolder:   false,
			ParentID:   destParentVirtualID,
			Provider:   destAcc.Provider,
			AccountID:  destAcc.ID,
			PhysicalID: serializedDestPhysIDs,
			CreatedAt:  time.Now(),
			ModifiedAt: time.Now(),
		}

		_ = a.database.SaveFile(newFileRecord)
		_ = a.database.LogActivity(newFileRecord.ID, newFileRecord.Name, "transfer", fmt.Sprintf("Copied from %s to %s", srcAcc.DisplayName, destAcc.DisplayName))

		if a.uploadHub != nil {
			a.uploadHub.broadcastProgress(progressMessage{
				Type:     "transfer_progress",
				ID:       transferID,
				Filename: f.Name,
				Percent:  100,
			})
		}

		runtime.EventsEmit(a.ctx, "transfer_completed", map[string]string{
			"transferId": transferID,
			"filename":   f.Name,
		})
	}()

	return nil
}

func (a *App) sendTransferError(transferID string, filename string, err error) {
	if a.uploadHub != nil {
		a.uploadHub.broadcastProgress(progressMessage{
			Type:     "transfer_progress",
			ID:       transferID,
			Filename: filename,
			Percent:  0,
			Error:    err.Error(),
		})
	}
	runtime.EventsEmit(a.ctx, "transfer_failed", map[string]string{
		"transferId": transferID,
		"filename":   filename,
		"error":      err.Error(),
	})
}

// RemoteUploadFromURL downloads a file from the given direct URL and uploads it straight to the selected cloud account.
func (a *App) RemoteUploadFromURL(parentVirtualID string, targetAccountID string, downloadURL string) error {
	u, err := url.Parse(downloadURL) // wait, url is from "net/url", let's import "net/url"!
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	filename := filepath.Base(u.Path)
	if filename == "" || filename == "." || filename == "/" {
		filename = "downloaded_file"
	}

	accounts, _ := a.database.GetAccounts()
	var destAcc db.AccountRecord
	found := false
	for _, acc := range accounts {
		if acc.ID == targetAccountID {
			destAcc = acc
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("destination account not found")
	}
	if !destAcc.Active {
		return fmt.Errorf("destination account is inactive")
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("RemoteUploadFromURL goroutine panicked: %v", r)
			}
		}()
		uploadID := fmt.Sprintf("url-up-%d", time.Now().UnixNano())
		runtime.EventsEmit(a.ctx, "upload_started", map[string]string{
			"uploadId": uploadID,
			"filename": "[Remote] " + filename,
		})

		// Use an HTTP client with a reasonable timeout to avoid hanging indefinitely
		httpClient := &http.Client{Timeout: 30 * time.Minute}
		resp, err := httpClient.Get(downloadURL)
		if err != nil {
			a.sendUploadError(uploadID, "[Remote] "+filename, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			a.sendUploadError(uploadID, "[Remote] "+filename, fmt.Errorf("bad status code: %d", resp.StatusCode))
			return
		}

		size := resp.ContentLength
		if disp := resp.Header.Get("Content-Disposition"); disp != "" {
			if _, params, err := mime.ParseMediaType(disp); err == nil {
				if fn, ok := params["filename"]; ok && fn != "" {
					filename = fn
				}
			}
		}

		destPhysParentID, err := a.resolvePhysicalFolder(parentVirtualID, destAcc)
		if err != nil {
			a.sendUploadError(uploadID, "[Remote] "+filename, err)
			return
		}

		p, err := sync.FetchActiveProviderClient(a.database, destAcc, nil)
		if err != nil {
			a.sendUploadError(uploadID, "[Remote] "+filename, err)
			return
		}

		var tracker *progressTracker
		var trackedReader io.Reader = resp.Body
		if a.uploadHub != nil {
			tracker = newProgressTracker(a.uploadHub, uploadID, "[Remote] "+filename, size, "upload_progress")
			trackedReader = tracker.reader(resp.Body)
		}

		physID, err := p.UploadFile(destPhysParentID, filename, trackedReader, size)
		if err != nil {
			a.sendUploadError(uploadID, "[Remote] "+filename, err)
			return
		}

		destPhysicalMap := make(router.PhysicalIDsMap)
		destPhysicalMap[destAcc.ID] = physID
		serializedPhysIDs, _ := router.SerializePhysicalIDs(destPhysicalMap)

		newFileRecord := db.FileRecord{
			ID:         uuid.New().String(),
			Name:       filename,
			Size:       size,
			IsFolder:   false,
			ParentID:   parentVirtualID,
			Provider:   destAcc.Provider,
			AccountID:  destAcc.ID,
			PhysicalID: serializedPhysIDs,
			CreatedAt:  time.Now(),
			ModifiedAt: time.Now(),
		}

		_ = a.database.SaveFile(newFileRecord)
		_ = a.database.LogActivity(newFileRecord.ID, newFileRecord.Name, "upload", fmt.Sprintf("Uploaded remotely from URL: %s (%s)", downloadURL, formatBytes(newFileRecord.Size)))

		if a.uploadHub != nil {
			a.uploadHub.broadcastProgress(progressMessage{
				Type:     "upload_progress",
				ID:       uploadID,
				Filename: "[Remote] " + filename,
				Percent:  100,
			})
		}
		runtime.EventsEmit(a.ctx, "upload_completed", map[string]string{
			"uploadId": uploadID,
			"filename": "[Remote] " + filename,
		})
	}()

	return nil
}

// sendUploadError is a small helper to broadcast upload failures
func (a *App) sendUploadError(uploadID string, filename string, err error) {
	if a.uploadHub != nil {
		a.uploadHub.broadcastProgress(progressMessage{
			Type:     "upload_progress",
			ID:       uploadID,
			Filename: filename,
			Percent:  0,
			Error:    err.Error(),
		})
	}
	runtime.EventsEmit(a.ctx, "upload_failed", map[string]string{
		"uploadId": uploadID,
		"filename": filename,
		"error":    err.Error(),
	})
}
