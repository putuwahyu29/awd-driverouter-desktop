package router_test

import (
	"driverouter/backend/db"
	"driverouter/backend/router"
	"os"
	"testing"
)

// setupTestDB creates a temporary in-memory database for testing.
func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	// Use a temp file so we can test file-based operations
	f, err := os.CreateTemp("", "driverouter_test_*.db")
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

func makeAccount(id, provider string, used, total int64) db.AccountRecord {
	return db.AccountRecord{
		ID:           id,
		Provider:     provider,
		DisplayName:  "Test Account " + id,
		Email:        id + "@test.com",
		AccessToken:  "token-" + id,
		RefreshToken: "refresh-" + id,
		TokenExpiry:  "2099-01-01T00:00:00Z",
		UsedSpace:    used,
		TotalSpace:   total,
		Active:       true,
	}
}

func TestSelectTargetAccounts_NoAccounts(t *testing.T) {
	database := setupTestDB(t)
	_, _, err := router.SelectTargetAccounts(database, "")
	if err == nil {
		t.Error("Expected error when no accounts connected, got nil")
	}
}

func TestSelectTargetAccounts_RoundRobin(t *testing.T) {
	database := setupTestDB(t)
	_ = database.SaveSetting("upload_strategy", "round_robin")

	acc1 := makeAccount("acc1", "google", 0, 15*1024*1024*1024)
	acc2 := makeAccount("acc2", "onedrive", 0, 5*1024*1024*1024)
	_ = database.SaveAccount(acc1)
	_ = database.SaveAccount(acc2)

	// First call
	targets1, strategy, err := router.SelectTargetAccounts(database, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if strategy != "round_robin" {
		t.Errorf("Expected strategy 'round_robin', got %q", strategy)
	}
	if len(targets1) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(targets1))
	}

	// Second call should pick a different account
	targets2, _, err := router.SelectTargetAccounts(database, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if targets1[0].ID == targets2[0].ID {
		t.Error("Round-robin should alternate accounts on successive calls")
	}
}

func TestSelectTargetAccounts_Mirror(t *testing.T) {
	database := setupTestDB(t)
	_ = database.SaveSetting("upload_strategy", "mirror")

	_ = database.SaveAccount(makeAccount("acc1", "google", 0, 15*1024*1024*1024))
	_ = database.SaveAccount(makeAccount("acc2", "onedrive", 0, 5*1024*1024*1024))
	_ = database.SaveAccount(makeAccount("acc3", "dropbox", 0, 2*1024*1024*1024))

	targets, strategy, err := router.SelectTargetAccounts(database, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if strategy != "mirror" {
		t.Errorf("Expected strategy 'mirror', got %q", strategy)
	}
	if len(targets) != 3 {
		t.Errorf("Mirror should return all 3 accounts, got %d", len(targets))
	}
}

func TestSelectTargetAccounts_MaxFree(t *testing.T) {
	database := setupTestDB(t)
	_ = database.SaveSetting("upload_strategy", "max_free")

	_ = database.SaveAccount(makeAccount("acc1", "google", 10*1024*1024*1024, 15*1024*1024*1024)) // 5GB free
	_ = database.SaveAccount(makeAccount("acc2", "onedrive", 1*1024*1024*1024, 5*1024*1024*1024)) // 4GB free

	targets, _, err := router.SelectTargetAccounts(database, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("Expected 1 target, got %d", len(targets))
	}
	if targets[0].ID != "acc1" {
		t.Errorf("Expected acc1 (most free space), got %s", targets[0].ID)
	}
}

func TestSelectTargetAccounts_LeastUsed(t *testing.T) {
	database := setupTestDB(t)
	_ = database.SaveSetting("upload_strategy", "least_used")

	_ = database.SaveAccount(makeAccount("acc1", "google", 14*1024*1024*1024, 15*1024*1024*1024)) // 93% used
	_ = database.SaveAccount(makeAccount("acc2", "onedrive", 1*1024*1024*1024, 5*1024*1024*1024)) // 20% used

	targets, _, err := router.SelectTargetAccounts(database, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if targets[0].ID != "acc2" {
		t.Errorf("Expected acc2 (least usage ratio), got %s", targets[0].ID)
	}
}

func TestSelectTargetAccounts_Manual(t *testing.T) {
	database := setupTestDB(t)

	_ = database.SaveAccount(makeAccount("acc1", "google", 0, 15*1024*1024*1024))
	_ = database.SaveAccount(makeAccount("acc2", "onedrive", 0, 5*1024*1024*1024))

	targets, strategy, err := router.SelectTargetAccounts(database, "acc2")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if strategy != "manual" {
		t.Errorf("Expected strategy 'manual', got %q", strategy)
	}
	if targets[0].ID != "acc2" {
		t.Errorf("Expected acc2 as manual target, got %s", targets[0].ID)
	}
}

func TestSelectTargetAccounts_ManualInvalidAccount(t *testing.T) {
	database := setupTestDB(t)
	_ = database.SaveAccount(makeAccount("acc1", "google", 0, 15*1024*1024*1024))

	_, _, err := router.SelectTargetAccounts(database, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent manual account, got nil")
	}
}

func TestSerializeDeserializePhysicalIDs(t *testing.T) {
	original := router.PhysicalIDsMap{
		"acc1": "physid-abc123",
		"acc2": "physid-def456",
	}

	serialized, err := router.SerializePhysicalIDs(original)
	if err != nil {
		t.Fatalf("SerializePhysicalIDs failed: %v", err)
	}

	deserialized, err := router.DeserializePhysicalIDs(serialized)
	if err != nil {
		t.Fatalf("DeserializePhysicalIDs failed: %v", err)
	}

	for k, v := range original {
		if deserialized[k] != v {
			t.Errorf("Roundtrip mismatch for key %q: got %q, want %q", k, deserialized[k], v)
		}
	}
}

func TestDeserializePhysicalIDs_Empty(t *testing.T) {
	result, err := router.DeserializePhysicalIDs("")
	if err != nil {
		t.Fatalf("Unexpected error for empty string: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map, got %v", result)
	}
}
