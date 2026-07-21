package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"driverouter/backend/db"
	"driverouter/backend/router"
	"driverouter/backend/sync"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// CompressFilesToZip downloads multiple virtual files, zips them locally into a temp file, and uploads the ZIP archive to the target account.
func (a *App) CompressFilesToZip(virtualIDs []string, parentVirtualID string, archiveName string) error {
	if len(virtualIDs) == 0 {
		return fmt.Errorf("no files selected for compression")
	}

	if !strings.HasSuffix(strings.ToLower(archiveName), ".zip") {
		archiveName += ".zip"
	}

	var filesToZip []db.FileRecord
	for _, id := range virtualIDs {
		f, err := a.database.GetFile(id)
		if err != nil {
			return err
		}
		if f.IsFolder {
			continue
		}
		filesToZip = append(filesToZip, f)
	}

	if len(filesToZip) == 0 {
		return fmt.Errorf("no valid files selected (folders are not supported for compression yet)")
	}

	var destAcc db.AccountRecord
	foundAcc := false
	accounts, _ := a.database.GetAccounts()

	for _, f := range filesToZip {
		for _, acc := range accounts {
			if acc.ID == f.AccountID && acc.Active {
				destAcc = acc
				foundAcc = true
				break
			}
		}
		if foundAcc {
			break
		}
	}

	if !foundAcc {
		for _, acc := range accounts {
			if acc.Active {
				destAcc = acc
				foundAcc = true
				break
			}
		}
	}

	if !foundAcc {
		return fmt.Errorf("no active cloud account found to upload the ZIP to")
	}

	go func() {
		uploadID := fmt.Sprintf("zip-up-%d", time.Now().UnixNano())
		runtime.EventsEmit(a.ctx, "upload_started", map[string]string{
			"uploadId": uploadID,
			"filename": archiveName,
		})

		tempFile, err := os.CreateTemp("", "driverouter-zip-*.zip")
		if err != nil {
			a.sendUploadError(uploadID, archiveName, fmt.Errorf("failed to create temporary archive: %w", err))
			return
		}
		tempFilePath := tempFile.Name()
		defer func() {
			tempFile.Close()
			os.Remove(tempFilePath)
		}()

		zipWriter := zip.NewWriter(tempFile)

		for idx, f := range filesToZip {
			physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
			if err != nil || len(physicalMap) == 0 {
				continue
			}
			var srcAcc db.AccountRecord
			var physID string
			foundSrc := false
			for accID, pID := range physicalMap {
				for _, acc := range accounts {
					if acc.ID == accID && acc.Active {
						srcAcc = acc
						physID = pID
						foundSrc = true
						break
					}
				}
				if foundSrc {
					break
				}
			}

			if !foundSrc {
				continue
			}

			percent := int((float64(idx) / float64(len(filesToZip))) * 50.0)
			if a.uploadHub != nil {
				a.uploadHub.broadcastProgress(progressMessage{
					Type:     "upload_progress",
					ID:       uploadID,
					Filename: fmt.Sprintf("Packing %s (%d/%d)...", f.Name, idx+1, len(filesToZip)),
					Percent:  percent,
				})
			}

			pSrc, err := sync.FetchActiveProviderClient(a.database, srcAcc, nil)
			if err != nil {
				continue
			}

			srcReader, err := pSrc.DownloadFile(physID)
			if err != nil {
				continue
			}

			zipEntryWriter, err := zipWriter.Create(f.Name)
			if err != nil {
				srcReader.Close()
				continue
			}

			_, _ = io.Copy(zipEntryWriter, srcReader)
			srcReader.Close()
		}

		zipWriter.Close()
		tempFile.Seek(0, 0)

		fileInfo, err := tempFile.Stat()
		if err != nil {
			a.sendUploadError(uploadID, archiveName, err)
			return
		}
		archiveSize := fileInfo.Size()

		destPhysParentID, err := a.resolvePhysicalFolder(parentVirtualID, destAcc)
		if err != nil {
			a.sendUploadError(uploadID, archiveName, err)
			return
		}

		pDest, err := sync.FetchActiveProviderClient(a.database, destAcc, nil)
		if err != nil {
			a.sendUploadError(uploadID, archiveName, err)
			return
		}

		var tracker *progressTracker
		var trackedReader io.Reader = tempFile
		if a.uploadHub != nil {
			tracker = newProgressTracker(a.uploadHub, uploadID, "Uploading "+archiveName, archiveSize, "upload_progress")
			trackedReader = tracker.reader(tempFile)
		}

		physID, err := pDest.UploadFile(destPhysParentID, archiveName, trackedReader, archiveSize)
		if err != nil {
			a.sendUploadError(uploadID, archiveName, err)
			return
		}

		destPhysicalMap := make(router.PhysicalIDsMap)
		destPhysicalMap[destAcc.ID] = physID
		serializedPhysIDs, _ := router.SerializePhysicalIDs(destPhysicalMap)

		newFileRecord := db.FileRecord{
			ID:         uuid.New().String(),
			Name:       archiveName,
			Size:       archiveSize,
			IsFolder:   false,
			ParentID:   parentVirtualID,
			Provider:   destAcc.Provider,
			AccountID:  destAcc.ID,
			PhysicalID: serializedPhysIDs,
			CreatedAt:  time.Now(),
			ModifiedAt: time.Now(),
		}

		_ = a.database.SaveFile(newFileRecord)
		_ = a.database.LogActivity(newFileRecord.ID, newFileRecord.Name, "zip", fmt.Sprintf("Compressed %d files into ZIP archive (%s)", len(filesToZip), formatBytes(newFileRecord.Size)))

		if a.uploadHub != nil {
			a.uploadHub.broadcastProgress(progressMessage{
				Type:     "upload_progress",
				ID:       uploadID,
				Filename: archiveName,
				Percent:  100,
			})
		}
		runtime.EventsEmit(a.ctx, "upload_completed", map[string]string{
			"uploadId": uploadID,
			"filename": archiveName,
		})
	}()

	return nil
}

// ExtractZipFile downloads a remote zip file, extracts it locally, uploads the extracted contents to the same parent directory, and cleans up.
func (a *App) ExtractZipFile(virtualZipID string, destParentVirtualID string) error {
	zipFileRecord, err := a.database.GetFile(virtualZipID)
	if err != nil {
		return err
	}

	physicalMap, err := router.DeserializePhysicalIDs(zipFileRecord.PhysicalID)
	if err != nil || len(physicalMap) == 0 {
		return fmt.Errorf("no physical copies found for the archive")
	}

	var srcAcc db.AccountRecord
	var physID string
	foundSrc := false
	accounts, _ := a.database.GetAccounts()
	for accID, pID := range physicalMap {
		for _, acc := range accounts {
			if acc.ID == accID && acc.Active {
				srcAcc = acc
				physID = pID
				foundSrc = true
				break
			}
		}
		if foundSrc {
			break
		}
	}

	if !foundSrc {
		return fmt.Errorf("no active account holds a copy of this archive")
	}

	go func() {
		uploadID := fmt.Sprintf("unzip-%d", time.Now().UnixNano())
		runtime.EventsEmit(a.ctx, "upload_started", map[string]string{
			"uploadId": uploadID,
			"filename": "Extracting " + zipFileRecord.Name,
		})

		pSrc, err := sync.FetchActiveProviderClient(a.database, srcAcc, nil)
		if err != nil {
			a.sendUploadError(uploadID, "Extracting "+zipFileRecord.Name, err)
			return
		}

		zipStream, err := pSrc.DownloadFile(physID)
		if err != nil {
			a.sendUploadError(uploadID, "Extracting "+zipFileRecord.Name, err)
			return
		}
		defer zipStream.Close()

		tempZipFile, err := os.CreateTemp("", "driverouter-extract-*.zip")
		if err != nil {
			a.sendUploadError(uploadID, "Extracting "+zipFileRecord.Name, err)
			return
		}
		tempZipPath := tempZipFile.Name()
		defer os.Remove(tempZipPath)

		_, err = io.Copy(tempZipFile, zipStream)
		tempZipFile.Close()
		if err != nil {
			a.sendUploadError(uploadID, "Extracting "+zipFileRecord.Name, err)
			return
		}

		zipReader, err := zip.OpenReader(tempZipPath)
		if err != nil {
			a.sendUploadError(uploadID, "Extracting "+zipFileRecord.Name, err)
			return
		}
		defer zipReader.Close()

		totalFiles := len(zipReader.File)
		for idx, file := range zipReader.File {
			if file.FileInfo().IsDir() {
				continue
			}

			percent := int((float64(idx) / float64(totalFiles)) * 100.0)
			if a.uploadHub != nil {
				a.uploadHub.broadcastProgress(progressMessage{
					Type:     "upload_progress",
					ID:       uploadID,
					Filename: fmt.Sprintf("Extracting: %s (%d/%d)", file.Name, idx+1, totalFiles),
					Percent:  percent,
				})
			}

			rc, err := file.Open()
			if err != nil {
				continue
			}

			destPhysParentID, err := a.resolvePhysicalFolder(destParentVirtualID, srcAcc)
			if err != nil {
				rc.Close()
				continue
			}

			pDest, err := sync.FetchActiveProviderClient(a.database, srcAcc, nil)
			if err != nil {
				rc.Close()
				continue
			}

			fileSize := int64(file.UncompressedSize64)
			entryPhysID, err := pDest.UploadFile(destPhysParentID, file.Name, rc, fileSize)
			rc.Close()
			if err != nil {
				continue
			}

			destPhysicalMap := make(router.PhysicalIDsMap)
			destPhysicalMap[srcAcc.ID] = entryPhysID
			serializedPhysIDs, _ := router.SerializePhysicalIDs(destPhysicalMap)

			newFileRecord := db.FileRecord{
				ID:         uuid.New().String(),
				Name:       file.Name,
				Size:       fileSize,
				IsFolder:   false,
				ParentID:   destParentVirtualID,
				Provider:   srcAcc.Provider,
				AccountID:  srcAcc.ID,
				PhysicalID: serializedPhysIDs,
				CreatedAt:  time.Now(),
				ModifiedAt: time.Now(),
			}
			_ = a.database.SaveFile(newFileRecord)
		}

		if a.uploadHub != nil {
			a.uploadHub.broadcastProgress(progressMessage{
				Type:     "upload_progress",
				ID:       uploadID,
				Filename: "Extracted " + zipFileRecord.Name,
				Percent:  100,
			})
		}
		_ = a.database.LogActivity(zipFileRecord.ID, zipFileRecord.Name, "unzip", fmt.Sprintf("Extracted ZIP archive contents (%d files)", totalFiles))
		runtime.EventsEmit(a.ctx, "upload_completed", map[string]string{
			"uploadId": uploadID,
			"filename": "Extracted " + zipFileRecord.Name,
		})
	}()

	return nil
}
