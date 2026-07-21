package main

import (
	"fmt"

	"driverouter/backend/db"
	"driverouter/backend/provider"
	"driverouter/backend/router"
	"driverouter/backend/sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// GetFileWebURL returns the remote web URL of a file or folder for sharing/viewing in browser.
func (a *App) GetFileWebURL(virtualID string) (string, error) {
	if virtualID == "root" || virtualID == "" {
		return "", fmt.Errorf("root folder does not have a remote web link")
	}

	f, err := a.database.GetFile(virtualID)
	if err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	if f.Provider == "virtual" {
		return "", fmt.Errorf("virtual items do not have a remote web link")
	}

	physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
	if err != nil || len(physicalMap) == 0 {
		return "", fmt.Errorf("no physical copies linked to this file")
	}

	var activeAcc db.AccountRecord
	var physID string
	found := false
	accounts, _ := a.database.GetAccounts()

	for accID, pID := range physicalMap {
		for _, acc := range accounts {
			if acc.ID == accID && acc.Active {
				activeAcc = acc
				physID = pID
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return "", fmt.Errorf("no active account is currently linked to this file")
	}

	p, err := sync.FetchActiveProviderClient(a.database, activeAcc, nil)
	if err != nil {
		return "", fmt.Errorf("failed to fetch provider client: %w", err)
	}

	if linker, ok := p.(interface {
		GetWebURL(physicalID string, isFolder bool) (string, error)
	}); ok {
		urlStr, err := linker.GetWebURL(physID, f.IsFolder)
		if err == nil {
			f.Shared = true
			_ = a.database.SaveFile(f)
		}
		return urlStr, err
	}

	return "", fmt.Errorf("provider '%s' does not support retrieving web URLs yet", f.Provider)
}

// OpenFileInBrowser retrieves the file's web URL and opens it in the default system browser.
func (a *App) OpenFileInBrowser(virtualID string) error {
	urlStr, err := a.GetFileWebURL(virtualID)
	if err != nil {
		return err
	}
	runtime.BrowserOpenURL(a.ctx, urlStr)
	return nil
}

// GetFilePermissions retrieves the permissions of a file/folder from its cloud provider.
func (a *App) GetFilePermissions(virtualID string) ([]provider.SharePermission, error) {
	f, err := a.database.GetFile(virtualID)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	if f.Provider == "virtual" {
		return nil, fmt.Errorf("virtual items do not have remote permissions")
	}

	physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
	if err != nil || len(physicalMap) == 0 {
		return nil, fmt.Errorf("no physical copies linked")
	}

	var activeAcc db.AccountRecord
	var physID string
	found := false
	accounts, _ := a.database.GetAccounts()

	for accID, pID := range physicalMap {
		for _, acc := range accounts {
			if acc.ID == accID && acc.Active {
				activeAcc = acc
				physID = pID
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("no active account linked")
	}

	p, err := sync.FetchActiveProviderClient(a.database, activeAcc, nil)
	if err != nil {
		return nil, err
	}

	if manager, ok := p.(provider.ShareManager); ok {
		return manager.ListPermissions(physID)
	}

	return nil, fmt.Errorf("provider '%s' does not support sharing options yet", f.Provider)
}

// AddFilePermission shares a file with a user email address.
func (a *App) AddFilePermission(virtualID, email, role string) error {
	f, err := a.database.GetFile(virtualID)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if f.Provider == "virtual" {
		return fmt.Errorf("cannot share virtual items")
	}

	physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
	if err != nil || len(physicalMap) == 0 {
		return fmt.Errorf("no physical copies linked")
	}

	var activeAcc db.AccountRecord
	var physID string
	found := false
	accounts, _ := a.database.GetAccounts()

	for accID, pID := range physicalMap {
		for _, acc := range accounts {
			if acc.ID == accID && acc.Active {
				activeAcc = acc
				physID = pID
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return fmt.Errorf("no active account linked")
	}

	p, err := sync.FetchActiveProviderClient(a.database, activeAcc, nil)
	if err != nil {
		return err
	}

	if manager, ok := p.(provider.ShareManager); ok {
		err = manager.AddPermission(physID, email, role)
		if err == nil {
			f.Shared = true
			_ = a.database.SaveFile(f)
			_ = a.database.LogActivity(f.ID, f.Name, "share", fmt.Sprintf("Shared with %s as %s", email, role))
		}
		return err
	}

	return fmt.Errorf("provider '%s' does not support sharing options yet", f.Provider)
}

// DeleteFilePermission revokes permission access for a specific permission ID.
func (a *App) DeleteFilePermission(virtualID, permID string) error {
	f, err := a.database.GetFile(virtualID)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if f.Provider == "virtual" {
		return fmt.Errorf("cannot modify virtual item permissions")
	}

	physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
	if err != nil || len(physicalMap) == 0 {
		return fmt.Errorf("no physical copies linked")
	}

	var activeAcc db.AccountRecord
	var physID string
	found := false
	accounts, _ := a.database.GetAccounts()

	for accID, pID := range physicalMap {
		for _, acc := range accounts {
			if acc.ID == accID && acc.Active {
				activeAcc = acc
				physID = pID
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return fmt.Errorf("no active account linked")
	}

	p, err := sync.FetchActiveProviderClient(a.database, activeAcc, nil)
	if err != nil {
		return err
	}

	if manager, ok := p.(provider.ShareManager); ok {
		err = manager.DeletePermission(physID, permID)
		if err == nil {
			_ = a.database.LogActivity(f.ID, f.Name, "share", "Revoked permission access")
			if perms, errList := manager.ListPermissions(physID); errList == nil {
				hasShares := false
				for _, pm := range perms {
					if pm.Role != "owner" {
						hasShares = true
						break
					}
				}
				f.Shared = hasShares
				_ = a.database.SaveFile(f)
			}
		}
		return err
	}

	return fmt.Errorf("provider '%s' does not support sharing options yet", f.Provider)
}

// SetFileGeneralAccess updates the general link access policy (restricted vs anyone).
func (a *App) SetFileGeneralAccess(virtualID, accessType, role string) error {
	f, err := a.database.GetFile(virtualID)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if f.Provider == "virtual" {
		return fmt.Errorf("cannot modify virtual item permissions")
	}

	physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
	if err != nil || len(physicalMap) == 0 {
		return fmt.Errorf("no physical copies linked")
	}

	var activeAcc db.AccountRecord
	var physID string
	found := false
	accounts, _ := a.database.GetAccounts()

	for accID, pID := range physicalMap {
		for _, acc := range accounts {
			if acc.ID == accID && acc.Active {
				activeAcc = acc
				physID = pID
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return fmt.Errorf("no active account linked")
	}

	p, err := sync.FetchActiveProviderClient(a.database, activeAcc, nil)
	if err != nil {
		return err
	}

	if manager, ok := p.(provider.ShareManager); ok {
		err = manager.SetGeneralAccess(physID, accessType, role)
		if err == nil {
			if accessType == "anyone" {
				f.Shared = true
				_ = a.database.LogActivity(f.ID, f.Name, "share", fmt.Sprintf("General access set to 'Anyone with link' (%s)", role))
			} else {
				_ = a.database.LogActivity(f.ID, f.Name, "share", "General access set to 'Restricted'")
				if perms, errList := manager.ListPermissions(physID); errList == nil {
					hasShares := false
					for _, pm := range perms {
						if pm.Role != "owner" {
							hasShares = true
							break
						}
					}
					f.Shared = hasShares
				}
			}
			_ = a.database.SaveFile(f)
		}
		return err
	}

	return fmt.Errorf("provider '%s' does not support sharing options yet", f.Provider)
}

// GetFileActivities returns log records tracking changes made to a file.
func (a *App) GetFileActivities(virtualID string) ([]db.ActivityRecord, error) {
	return a.database.GetFileActivities(virtualID)
}

// GetGeneralActivities returns a list of recent activities globally.
func (a *App) GetGeneralActivities(limit int) ([]db.ActivityRecord, error) {
	return a.database.GetGeneralActivities(limit)
}
