package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"driverouter/backend/db"
	"driverouter/backend/provider"
	"driverouter/backend/router"
	"driverouter/backend/sync"

	"github.com/google/uuid"
)

// GetFiles lists virtual files/folders inside a parent ID, updating metadata in real-time.
func (a *App) GetFiles(parentID string, starred bool, search string) ([]db.FileRecord, error) {
	if parentID == "__shared__" {
		return a.getSharedFiles(search)
	}

	if parentID == "" {
		parentID = "root"
	}

	// Trigger real-time remote sync for the directory listing
	if !starred && search == "" {
		// Clean up files in the database that are marked as 'shared' but might have been moved to root by reconciliation
		// or vice-versa, to ensure My Drive and Shared views stay separate.
		// Note: Shared files from getSharedFiles are dynamic, but if they ever got saved to DB, we must filter them.

		accounts, err := a.database.GetAccounts()
		if err == nil {
			var activeAccounts []db.AccountRecord
			for _, acc := range accounts {
				if acc.Active {
					activeAccounts = append(activeAccounts, acc)
				}
			}

			// Gather remote items from active accounts
			type remoteItem struct {
				meta  provider.FileMetadata
				accID string
			}
			var allRemoteItems []remoteItem
			queriedAccounts := make(map[string]bool)

			if parentID == "root" {
				for _, acc := range activeAccounts {
					p, err := sync.FetchActiveProviderClient(a.database, acc, nil)
					if err == nil {
						items, err := p.ListDirectory("root")
						if err == nil {
							queriedAccounts[acc.ID] = true
							for _, item := range items {
								allRemoteItems = append(allRemoteItems, remoteItem{meta: item, accID: acc.ID})
							}
						}
					}
				}
			} else {
				vFolder, err := a.database.GetFile(parentID)
				if err == nil {
					physicalMap, err := router.DeserializePhysicalIDs(vFolder.PhysicalID)
					if err == nil {
						for _, acc := range activeAccounts {
							if physFolderID, exists := physicalMap[acc.ID]; exists {
								p, err := sync.FetchActiveProviderClient(a.database, acc, nil)
								if err == nil {
									items, err := p.ListDirectory(physFolderID)
									if err == nil {
										queriedAccounts[acc.ID] = true
										for _, item := range items {
											allRemoteItems = append(allRemoteItems, remoteItem{meta: item, accID: acc.ID})
										}
									}
								}
							}
						}
					}
				}
			}

			// Reconcile remote items with database
			localFiles, err := a.database.GetFiles(parentID, false, "")
			if err == nil {
				localMap := make(map[string]db.FileRecord)
				for _, lf := range localFiles {
					key := fmt.Sprintf("%s_%t", lf.Name, lf.IsFolder)
					localMap[key] = lf
				}

				seenLocalIDs := make(map[string]bool)

				// Process remote items
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
						// Self-healing: Check if the file already exists in the database under a different parent ID
						// Only heal if the existing record is NOT a shared file, to prevent shared files leaking into My Drive.
						existingGlobal, err := a.database.FindFileByPhysicalID(ri.accID, ri.meta.PhysicalID)
						if err == nil && existingGlobal.ParentID != "shared" {
							// Found! Align parent_id to the current parentID and update details
							existingGlobal.ParentID = parentID
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
								ParentID:   parentID,
								Provider:   ri.meta.Provider,
								AccountID:  ri.accID,
								PhysicalID: ser,
								CreatedAt:  ri.meta.CreatedAt,
								ModifiedAt: ri.meta.ModifiedAt,
								Starred:    false,
							}
							_ = a.database.SaveFile(rec)
							// Update local map to prevent duplicates
							localMap[key] = rec
							seenLocalIDs[newID] = true
						}
					}
				}

				// Remove stale listings for queried accounts
				for _, lf := range localFiles {
					if !seenLocalIDs[lf.ID] {
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
			}
		}
	}

	return a.database.GetFiles(parentID, starred, search)
}

// CreateFolder creates a virtual folder.
func (a *App) CreateFolder(parentID, name string) (string, error) {
	if parentID == "" {
		parentID = "root"
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
