package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	gosync "sync"
	"time"

	"driverouter/backend/db"
	"driverouter/backend/provider"
	"driverouter/backend/router"
	"driverouter/backend/sync"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// GetFiles lists virtual files/folders inside a parent ID immediately from local index, while updating metadata asynchronously in the background.
func (a *App) GetFiles(parentID string, starred bool, search string) ([]db.FileRecord, error) {
	if parentID == "__shared__" {
		return a.getSharedFiles(search)
	}

	if parentID == "" {
		parentID = "root"
	}

	// Trigger async background remote sync for directory reconciliation so UI never blocks
	if !starred && search == "" {
		go func(pID string) {
			accounts, err := a.database.GetAccounts()
			if err != nil {
				return
			}
			var activeAccounts []db.AccountRecord
			for _, acc := range accounts {
				if acc.Active {
					activeAccounts = append(activeAccounts, acc)
				}
			}

			type remoteItem struct {
				meta  provider.FileMetadata
				accID string
			}
			var allRemoteItems []remoteItem
			queriedAccounts := make(map[string]bool)

			if pID == "root" {
				var wg gosync.WaitGroup
				var mu gosync.Mutex
				for _, acc := range activeAccounts {
					wg.Add(1)
					go func(acc db.AccountRecord) {
						defer wg.Done()
						p, err := sync.FetchActiveProviderClient(a.database, acc, nil)
						if err == nil {
							items, err := p.ListDirectory("root")
							if err == nil {
								mu.Lock()
								queriedAccounts[acc.ID] = true
								for _, item := range items {
									allRemoteItems = append(allRemoteItems, remoteItem{meta: item, accID: acc.ID})
								}
								mu.Unlock()
							}
						}
					}(acc)
				}
				wg.Wait()
			} else {
				vFolder, err := a.database.GetFile(pID)
				if err == nil {
					physicalMap, err := router.DeserializePhysicalIDs(vFolder.PhysicalID)
					if err == nil {
						var wg gosync.WaitGroup
						var mu gosync.Mutex
						for _, acc := range activeAccounts {
							if physFolderID, exists := physicalMap[acc.ID]; exists {
								wg.Add(1)
								go func(acc db.AccountRecord, physFolderID string) {
									defer wg.Done()
									p, err := sync.FetchActiveProviderClient(a.database, acc, nil)
									if err == nil {
										items, err := p.ListDirectory(physFolderID)
										if err == nil {
											mu.Lock()
											queriedAccounts[acc.ID] = true
											for _, item := range items {
												allRemoteItems = append(allRemoteItems, remoteItem{meta: item, accID: acc.ID})
											}
											mu.Unlock()
										}
									}
								}(acc, physFolderID)
							}
						}
						wg.Wait()
					}
				}
			}

			// Reconcile remote items with local database asynchronously
			localFiles, err := a.database.GetFiles(pID, false, "")
			if err == nil {
				localMap := make(map[string]db.FileRecord)
				for _, lf := range localFiles {
					key := fmt.Sprintf("%s_%t", lf.Name, lf.IsFolder)
					localMap[key] = lf
				}

				seenLocalIDs := make(map[string]bool)

				for _, ri := range allRemoteItems {
					key := fmt.Sprintf("%s_%t", ri.meta.Name, ri.meta.IsFolder)
					existing, found := localMap[key]

					if found {
						seenLocalIDs[existing.ID] = true
						physMap, _ := router.DeserializePhysicalIDs(existing.PhysicalID)
						if physMap == nil {
							physMap = make(router.PhysicalIDsMap)
						}

						if physMap[ri.accID] != ri.meta.PhysicalID {
							physMap[ri.accID] = ri.meta.PhysicalID
							ser, _ := router.SerializePhysicalIDs(physMap)
							existing.PhysicalID = ser
							existing.Size = ri.meta.Size
							existing.ModifiedAt = ri.meta.ModifiedAt
							_ = a.database.SaveFile(existing)
						}
					} else {
						existingGlobal, err := a.database.FindFileByPhysicalID(ri.accID, ri.meta.PhysicalID)
						if err == nil && existingGlobal.ParentID != "shared" {
							existingGlobal.ParentID = pID
							existingGlobal.Size = ri.meta.Size
							existingGlobal.ModifiedAt = ri.meta.ModifiedAt

							physMap, _ := router.DeserializePhysicalIDs(existingGlobal.PhysicalID)
							if physMap == nil {
								physMap = make(router.PhysicalIDsMap)
							}
							physMap[ri.accID] = ri.meta.PhysicalID
							ser, _ := router.SerializePhysicalIDs(physMap)
							existingGlobal.PhysicalID = ser

							_ = a.database.SaveFile(existingGlobal)
							seenLocalIDs[existingGlobal.ID] = true
							localMap[key] = existingGlobal
						} else {
							newID := uuid.New().String()
							physMap := router.PhysicalIDsMap{ri.accID: ri.meta.PhysicalID}
							ser, _ := router.SerializePhysicalIDs(physMap)
							rec := db.FileRecord{
								ID:         newID,
								Name:       ri.meta.Name,
								Size:       ri.meta.Size,
								IsFolder:   ri.meta.IsFolder,
								ParentID:   pID,
								Provider:   ri.meta.Provider,
								AccountID:  ri.accID,
								PhysicalID: ser,
								CreatedAt:  ri.meta.CreatedAt,
								ModifiedAt: ri.meta.ModifiedAt,
								Starred:    false,
							}
							_ = a.database.SaveFile(rec)
							localMap[key] = rec
							seenLocalIDs[newID] = true
						}
					}
				}

				for _, lf := range localFiles {
					if !seenLocalIDs[lf.ID] {
						// Do not auto-delete virtual items (e.g. newly created folders or pending virtual files) that have no physical mapping yet
						if lf.Provider == "virtual" || lf.PhysicalID == "" {
							continue
						}
						physMap, _ := router.DeserializePhysicalIDs(lf.PhysicalID)
						if physMap != nil {
							updated := false
							for accID := range queriedAccounts {
								if _, exists := physMap[accID]; exists {
									delete(physMap, accID)
									updated = true
								}
							}
							if len(physMap) == 0 {
								_ = a.database.DeleteFile(lf.ID)
							} else if updated {
								ser, _ := router.SerializePhysicalIDs(physMap)
								lf.PhysicalID = ser
								_ = a.database.SaveFile(lf)
							}
						} else {
							_ = a.database.DeleteFile(lf.ID)
						}
					}
				}

				// Emit event to frontend to refresh UI seamlessly when new files/updates arrive
				if a.ctx != nil {
					runtime.EventsEmit(a.ctx, "files-updated", parentID)
				}
			}
		}(parentID)
	}

	return a.database.GetFiles(parentID, starred, search)
}

// CreateFolder creates a virtual folder.
func (a *App) CreateFolder(parentID, name string) (string, error) {
	if parentID == "" {
		parentID = "root"
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = "Folder Baru"
	}

	id := uuid.New().String()
	record := db.FileRecord{
		ID:         id,
		Name:       name,
		Size:       0,
		IsFolder:   true,
		ParentID:   parentID,
		Provider:   "virtual",
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
	}

	err := a.database.SaveFile(record)
	if err != nil {
		return "", err
	}

	_ = a.database.LogActivity(record.ID, record.Name, "create_folder", fmt.Sprintf("Membuat folder baru: '%s'", record.Name))
	return id, nil
}

// RenameFile renames a virtual file or folder.
func (a *App) RenameFile(virtualID, newName string) error {
	// First, fetch record
	f, err := a.database.GetFile(virtualID)
	if err != nil {
		return err
	}

	// Rename physically across all clouds containing this file
	physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
	if err == nil && len(physicalMap) > 0 {
		for accID, physID := range physicalMap {
			acc, err := a.getAccount(accID)
			if err != nil {
				continue
			}

			p, err := sync.FetchActiveProviderClient(a.database, acc, nil)
			if err != nil {
				continue
			}

			// Some clouds rename via metadata patches or simple renames.
			// Google and OneDrive support PATCH / DELETE/ PUT.
			// To keep it simple, we can delete/upload or update.
			// For folders we just update locally.
			// For files, we update the metadata.
			// Let's implement rename inside adapters or just let it update in our local virtual tree.
			// To be accurate, we rename the file locally. Physical renames are optional but we can add
			// physical rename if the API provides it.
			// Google Drive: PATCH https://www.googleapis.com/drive/v3/files/{fileId} with {"name": "newName"}
			// OneDrive: PATCH https://graph.microsoft.com/v1.0/me/drive/items/{itemId} with {"name": "newName"}
			// Dropbox: POST https://api.dropboxapi.com/2/files/move_v2 with {"from_path": "id:...", "to_path": "/newpath"}
			// Let's trigger physical renames:
			go func(providerName, physicalID, newName string, providerClient provider.Provider) {
				// Inline physical renaming logic
				switch providerName {
				case "google":
					if g, ok := providerClient.(*provider.GoogleDriveProvider); ok {
						meta := map[string]interface{}{"name": newName}
						bodyBytes, _ := json.Marshal(meta)
						_, _ = g.Client.Post(fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s", physicalID), "application/json", bytes.NewBuffer(bodyBytes))
					}
				case "onedrive":
					if od, ok := providerClient.(*provider.OneDriveProvider); ok {
						meta := map[string]interface{}{"name": newName}
						bodyBytes, _ := json.Marshal(meta)
						req, err := http.NewRequest("PATCH", fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s", physicalID), bytes.NewBuffer(bodyBytes))
						if err == nil {
							req.Header.Set("Content-Type", "application/json")
							resp, err := od.Client.Do(req)
							if err == nil {
								resp.Body.Close()
							}
						}
					}
				}
			}(acc.Provider, physID, newName, p)
		}
	}

	errRename := a.database.UpdateFileName(virtualID, newName)
	if errRename == nil {
		_ = a.database.LogActivity(virtualID, newName, "rename", fmt.Sprintf("Renamed from '%s' to '%s'", f.Name, newName))
	}
	return errRename
}

// DeleteFile deletes a file physically and removes the database record.
func (a *App) DeleteFile(virtualID string) error {
	f, err := a.database.GetFile(virtualID)
	if err != nil {
		return err
	}

	// Resolve target file physical locations
	physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
	if err == nil && len(physicalMap) > 0 {
		for accID, physID := range physicalMap {
			acc, err := a.getAccount(accID)
			if err != nil {
				continue
			}

			p, err := sync.FetchActiveProviderClient(a.database, acc, nil)
			if err != nil {
				continue
			}

			err = p.DeleteFile(physID)
			if err != nil {
				log.Printf("Warning: failed to delete physical file %s on %s: %v", physID, acc.Provider, err)
			}
		}
	}

	isTrashSupported := false
	prov := f.Provider
	if prov == "google" || prov == "onedrive" || prov == "dropbox" || prov == "box" || prov == "yandex" || prov == "pcloud" || prov == "mega" {
		isTrashSupported = true
	}

	var errDel error
	if isTrashSupported {
		errDel = a.database.DeleteFile(virtualID)
		if errDel == nil {
			_ = a.database.LogActivity(virtualID, f.Name, "delete", fmt.Sprintf("Moved '%s' to trash", f.Name))
		}
	} else {
		errDel = a.database.PermanentlyDeleteFile(virtualID)
		if errDel == nil {
			_ = a.database.LogActivity(virtualID, f.Name, "delete", fmt.Sprintf("Permanently deleted '%s'", f.Name))
		}
	}
	return errDel
}

// GetTrashedFiles fetches all soft-deleted files from the database.
func (a *App) GetTrashedFiles() ([]db.FileRecord, error) {
	return a.database.GetTrashedFiles()
}

// RestoreFile restores a soft-deleted file/folder record back to My Drive.
func (a *App) RestoreFile(virtualID string) error {
	f, err := a.database.GetFile(virtualID)
	if err != nil {
		return err
	}
	errRest := a.database.RestoreFile(virtualID)
	if errRest == nil {
		_ = a.database.LogActivity(virtualID, f.Name, "restore", fmt.Sprintf("Restored '%s' from trash", f.Name))
	}
	return errRest
}

// PermanentlyDeleteFile physically deletes a file/folder record from the local database.
func (a *App) PermanentlyDeleteFile(virtualID string) error {
	f, err := a.database.GetFile(virtualID)
	if err != nil {
		return err
	}
	errDel := a.database.PermanentlyDeleteFile(virtualID)
	if errDel == nil {
		_ = a.database.LogActivity(virtualID, f.Name, "delete", fmt.Sprintf("Permanently deleted '%s'", f.Name))
	}
	return errDel
}

// GetVirtualFolders returns all virtual folder records from the database.
func (a *App) GetVirtualFolders() ([]db.FileRecord, error) {
	rows, err := a.database.Conn.Query("SELECT id, name, parent_id FROM files WHERE is_folder = 1 AND deleted = 0")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []db.FileRecord
	// Add root manually as the base directory option
	list = append(list, db.FileRecord{
		ID:       "root",
		Name:     "Main Storage root",
		IsFolder: true,
	})

	for rows.Next() {
		var f db.FileRecord
		if err := rows.Scan(&f.ID, &f.Name, &f.ParentID); err != nil {
			return nil, err
		}
		f.IsFolder = true
		// Skip adding root if it somehow gets returned from DB query
		if f.ID != "root" {
			list = append(list, f)
		}
	}
	return list, nil
}

// FindDuplicateFiles queries the indexed virtual database for files sharing the same name and size, suggesting redundancy.
func (a *App) FindDuplicateFiles() ([]db.FileRecord, error) {
	rows, err := a.database.Conn.Query(`
		SELECT name, size FROM files 
		WHERE is_folder = 0 AND deleted = 0 
		GROUP BY name, size 
		HAVING COUNT(*) > 1
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var duplicates []db.FileRecord

	for rows.Next() {
		var name string
		var size int64
		if err := rows.Scan(&name, &size); err != nil {
			return nil, err
		}

		subRows, err := a.database.Conn.Query(`
			SELECT id, name, size, parent_id, provider, account_id, physical_id, created_at, modified_at 
			FROM files 
			WHERE name = ? AND size = ? AND is_folder = 0 AND deleted = 0
		`, name, size)
		if err != nil {
			continue
		}

		for subRows.Next() {
			var f db.FileRecord
			var createdStr, modifiedStr string
			err := subRows.Scan(&f.ID, &f.Name, &f.Size, &f.ParentID, &f.Provider, &f.AccountID, &f.PhysicalID, &createdStr, &modifiedStr)
			if err != nil {
				subRows.Close()
				return nil, err
			}
			f.IsFolder = false
			f.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
			f.ModifiedAt, _ = time.Parse(time.RFC3339, modifiedStr)
			duplicates = append(duplicates, f)
		}
		subRows.Close()
	}

	return duplicates, nil
}
