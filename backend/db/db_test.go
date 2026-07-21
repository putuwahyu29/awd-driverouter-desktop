package db_test

import (
	"driverouter/backend/db"
	"fmt"
	"os"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	f, err := os.CreateTemp("", "driverouter_db_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db file: %v", err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	database, err := db.InitDBFromPath(f.Name())
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestInitDB_CreatesRootFolder(t *testing.T) {
	database := setupTestDB(t)
	f, err := database.GetFile("root")
	if err != nil {
		t.Fatalf("Expected root folder to exist: %v", err)
	}
	if !f.IsFolder {
		t.Error("Root should be a folder")
	}
	if f.Name != "My Drive" {
		t.Errorf("Expected root name 'My Drive', got %q", f.Name)
	}
}

func TestSaveAndGetAccount(t *testing.T) {
	database := setupTestDB(t)

	acc := db.AccountRecord{
		ID:           "test-acc-1",
		Provider:     "google",
		DisplayName:  "Test User",
		Email:        "test@gmail.com",
		AccessToken:  "access-token-plain",
		RefreshToken: "refresh-token-plain",
		TokenExpiry:  "2099-01-01T00:00:00Z",
		UsedSpace:    1024,
		TotalSpace:   15 * 1024 * 1024 * 1024,
		Active:       true,
	}

	err := database.SaveAccount(acc)
	if err != nil {
		t.Fatalf("SaveAccount failed: %v", err)
	}

	accounts, err := database.GetAccounts()
	if err != nil {
		t.Fatalf("GetAccounts failed: %v", err)
	}

	if len(accounts) != 1 {
		t.Fatalf("Expected 1 account, got %d", len(accounts))
	}

	retrieved := accounts[0]
	if retrieved.ID != acc.ID {
		t.Errorf("ID mismatch: got %q, want %q", retrieved.ID, acc.ID)
	}
	if retrieved.Provider != acc.Provider {
		t.Errorf("Provider mismatch: got %q, want %q", retrieved.Provider, acc.Provider)
	}
	// Tokens should be decrypted back to original values
	if retrieved.AccessToken != acc.AccessToken {
		t.Errorf("AccessToken mismatch after decrypt: got %q, want %q", retrieved.AccessToken, acc.AccessToken)
	}
	if retrieved.RefreshToken != acc.RefreshToken {
		t.Errorf("RefreshToken mismatch after decrypt: got %q, want %q", retrieved.RefreshToken, acc.RefreshToken)
	}
	if !retrieved.Active {
		t.Error("Expected account to be active")
	}
}

func TestSaveAndGetFile(t *testing.T) {
	database := setupTestDB(t)

	now := time.Now().Truncate(time.Second)
	f := db.FileRecord{
		ID:         "file-1",
		Name:       "test_document.pdf",
		Size:       1024 * 1024,
		IsFolder:   false,
		ParentID:   "root",
		Provider:   "google",
		AccountID:  "acc-1",
		PhysicalID: `{"acc-1":"physid-abc"}`,
		CreatedAt:  now,
		ModifiedAt: now,
	}

	err := database.SaveFile(f)
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	retrieved, err := database.GetFile("file-1")
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}

	if retrieved.Name != f.Name {
		t.Errorf("Name mismatch: got %q, want %q", retrieved.Name, f.Name)
	}
	if retrieved.Size != f.Size {
		t.Errorf("Size mismatch: got %d, want %d", retrieved.Size, f.Size)
	}
	if retrieved.IsFolder {
		t.Error("Expected IsFolder to be false")
	}
	if retrieved.ParentID != f.ParentID {
		t.Errorf("ParentID mismatch: got %q, want %q", retrieved.ParentID, f.ParentID)
	}
}

func TestSaveAndGetFiles_List(t *testing.T) {
	database := setupTestDB(t)

	now := time.Now()
	for i := 0; i < 5; i++ {
		_ = database.SaveFile(db.FileRecord{
			ID:         fmt.Sprintf("file-%d", i),
			Name:       fmt.Sprintf("file_%d.txt", i),
			Size:       int64(i) * 1024,
			IsFolder:   false,
			ParentID:   "root",
			Provider:   "google",
			AccountID:  "acc-1",
			PhysicalID: `{"acc-1":"phys"}`,
			CreatedAt:  now,
			ModifiedAt: now,
		})
	}

	files, err := database.GetFiles("root", false, "")
	if err != nil {
		t.Fatalf("GetFiles failed: %v", err)
	}

	// root folder itself is there + 5 files
	if len(files) < 5 {
		t.Errorf("Expected at least 5 files, got %d", len(files))
	}
}

func TestSoftDeleteAndRestore(t *testing.T) {
	database := setupTestDB(t)

	now := time.Now()
	_ = database.SaveFile(db.FileRecord{
		ID: "del-file-1", Name: "to_delete.txt", Size: 100,
		ParentID: "root", Provider: "google", AccountID: "acc-1",
		PhysicalID: `{"acc-1":"phys"}`, CreatedAt: now, ModifiedAt: now,
	})

	// Soft delete
	err := database.DeleteFile("del-file-1")
	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	// Should not appear in normal listing
	files, _ := database.GetFiles("root", false, "")
	for _, f := range files {
		if f.ID == "del-file-1" {
			t.Error("Deleted file should not appear in GetFiles")
		}
	}

	// Should appear in trash
	trashed, err := database.GetTrashedFiles()
	if err != nil {
		t.Fatalf("GetTrashedFiles failed: %v", err)
	}
	found := false
	for _, f := range trashed {
		if f.ID == "del-file-1" {
			found = true
		}
	}
	if !found {
		t.Error("Deleted file should appear in GetTrashedFiles")
	}

	// Restore it
	err = database.RestoreFile("del-file-1")
	if err != nil {
		t.Fatalf("RestoreFile failed: %v", err)
	}

	files, _ = database.GetFiles("root", false, "")
	restored := false
	for _, f := range files {
		if f.ID == "del-file-1" {
			restored = true
		}
	}
	if !restored {
		t.Error("Restored file should appear in GetFiles")
	}
}

func TestLogActivity_PrunesAt1000(t *testing.T) {
	database := setupTestDB(t)

	// Insert 1005 activity records
	for i := 0; i < 1005; i++ {
		_ = database.LogActivity("root", "testfile.txt", "test", fmt.Sprintf("entry %d", i))
	}

	activities, err := database.GetGeneralActivities(2000)
	if err != nil {
		t.Fatalf("GetGeneralActivities failed: %v", err)
	}

	if len(activities) > 1000 {
		t.Errorf("Activity log should be pruned to 1000, got %d", len(activities))
	}
}

func TestSaveSetting_GetSetting(t *testing.T) {
	database := setupTestDB(t)

	err := database.SaveSetting("test_key", "test_value")
	if err != nil {
		t.Fatalf("SaveSetting failed: %v", err)
	}

	val, err := database.GetSetting("test_key")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	if val != "test_value" {
		t.Errorf("GetSetting: got %q, want %q", val, "test_value")
	}
}

func TestGetSetting_MissingKey(t *testing.T) {
	database := setupTestDB(t)

	val, err := database.GetSetting("nonexistent_key")
	if err != nil {
		t.Fatalf("GetSetting for missing key should not error, got: %v", err)
	}
	if val != "" {
		t.Errorf("Missing setting should return empty string, got %q", val)
	}
}
