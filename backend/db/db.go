package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"driverouter/backend/security"
	_ "modernc.org/sqlite"
)

type FileRecord struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	IsFolder   bool      `json:"isFolder"`
	ParentID   string    `json:"parentId"`
	Provider   string    `json:"provider"`
	AccountID  string    `json:"accountId"`
	PhysicalID string    `json:"physicalId"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
	Starred    bool      `json:"starred"`
	Shared     bool      `json:"shared"`
}

type AccountRecord struct {
	ID           string `json:"id"`
	Provider     string `json:"provider"`
	DisplayName  string `json:"displayName"`
	Email        string `json:"email"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	TokenExpiry  string `json:"tokenExpiry"`
	UsedSpace    int64  `json:"usedSpace"`
	TotalSpace   int64  `json:"totalSpace"`
	Active       bool   `json:"active"`
}

type DB struct {
	Conn *sql.DB
	mu   sync.Mutex
}

// Exec executes a SQL write query with mutex serialization and retry with exponential backoff on SQLITE_BUSY.
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	var res sql.Result
	var err error
	for i := 0; i < 15; i++ {
		res, err = db.Conn.Exec(query, args...)
		if err == nil {
			return res, nil
		}
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "database is locked") || strings.Contains(errStr, "sqlite_busy") || strings.Contains(errStr, "locked") {
			time.Sleep(time.Duration(20*(i+1)) * time.Millisecond)
			continue
		}
		return res, err
	}
	return res, err
}

type SyncTask struct {
	ID             string `json:"id"`
	LocalPath      string `json:"localPath"`
	TargetFolderID string `json:"targetFolderId"`
	AccountID      string `json:"accountId"`
	SyncMode       string `json:"syncMode"`
	Enabled        bool   `json:"enabled"`
	LastSync       string `json:"lastSync"`
}

// InitDB initializes the SQLite database in the user's config directory.
func InitDB() (*DB, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	appDir := filepath.Join(configDir, "AwdDriveRouter")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create app data directory: %w", err)
	}
	dbPath := filepath.Join(appDir, "awd_driverouter.db")

	// Migration check: If target DB does not exist or is empty, auto-migrate from legacy paths
	shouldMigrate := false
	if info, err := os.Stat(dbPath); os.IsNotExist(err) || info.Size() == 0 {
		shouldMigrate = true
	} else {
		// Test if current database has 0 accounts
		tempDB, err := InitDBFromPath(dbPath)
		if err == nil {
			var count int
			_ = tempDB.Conn.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&count)
			if count == 0 {
				shouldMigrate = true
			}
			tempDB.Conn.Close()
		}
	}

	if shouldMigrate {
		oldPaths := []string{
			filepath.Join(configDir, "driverouter", "driverouter.db"),
			filepath.Join(configDir, "teledrive", "teledrive.db"),
			filepath.Join(configDir, "Teledrive", "teledrive.db"),
			filepath.Join(".", "teledrive.db"),
			filepath.Join(".", "driverouter.db"),
		}
		for _, old := range oldPaths {
			if info, err := os.Stat(old); err == nil && info.Size() > 0 {
				data, err := os.ReadFile(old)
				if err == nil && len(data) > 0 {
					_ = os.WriteFile(dbPath, data, 0644)
					log.Printf("[Migration] Successfully restored database from %s -> %s", old, dbPath)
					break
				}
			}
		}
	}

	return InitDBFromPath(dbPath)
}

// InitDBFromPath initializes the SQLite database at the given path.
// This is used by tests to create isolated temporary databases.
func InitDBFromPath(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=10000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	conn.SetMaxOpenConns(1) // Prevent concurrent write locks in SQLite

	// Explicitly configure WAL mode and busy timeout pragmas on connection
	_, _ = conn.Exec("PRAGMA journal_mode = WAL;")
	_, _ = conn.Exec("PRAGMA busy_timeout = 10000;")
	_, _ = conn.Exec("PRAGMA synchronous = NORMAL;")

	// Create tables if they don't exist
	queries := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			provider TEXT NOT NULL,
			display_name TEXT,
			email TEXT,
			access_token TEXT,
			refresh_token TEXT,
			token_expiry TEXT,
			used_space INTEGER,
			total_space INTEGER,
			active INTEGER DEFAULT 1
		);`,
		`CREATE TABLE IF NOT EXISTS files (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			size INTEGER,
			is_folder INTEGER,
			parent_id TEXT,
			provider TEXT,
			account_id TEXT,
			physical_id TEXT,
			created_at TEXT,
			modified_at TEXT,
			starred INTEGER DEFAULT 0,
			deleted INTEGER DEFAULT 0,
			shared INTEGER DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS activities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_id TEXT NOT NULL,
			file_name TEXT NOT NULL,
			action TEXT NOT NULL,
			details TEXT,
			timestamp TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sync_tasks (
			id TEXT PRIMARY KEY,
			local_path TEXT NOT NULL,
			target_folder_id TEXT NOT NULL,
			account_id TEXT NOT NULL,
			sync_mode TEXT NOT NULL,
			enabled INTEGER DEFAULT 1,
			last_sync TEXT
		);`,
	}

	for _, q := range queries {
		if _, err := conn.Exec(q); err != nil {
			return nil, fmt.Errorf("failed to execute migration query: %w", err)
		}
	}

	// Add shared column if it does not exist (for existing DBs)
	_, _ = conn.Exec("ALTER TABLE files ADD COLUMN shared INTEGER DEFAULT 0")

	// Create root directory in files table if not exist
	var exists bool
	err = conn.QueryRow("SELECT EXISTS(SELECT 1 FROM files WHERE id = 'root')").Scan(&exists)
	if err == nil && !exists {
		_, _ = conn.Exec(
			"INSERT INTO files (id, name, size, is_folder, parent_id, provider, account_id, physical_id, created_at, modified_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"root", "My Drive", 0, 1, "", "", "", "", time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339),
		)
	}

	// Set default strategy if not exist
	var strategyExists bool
	err = conn.QueryRow("SELECT EXISTS(SELECT 1 FROM settings WHERE key = 'upload_strategy')").Scan(&strategyExists)
	if err == nil && !strategyExists {
		_, _ = conn.Exec("INSERT INTO settings (key, value) VALUES ('upload_strategy', 'round_robin')")
	}

	// Clean up any orphaned files whose parents no longer exist in the files table
	_, _ = conn.Exec("DELETE FROM files WHERE parent_id != 'root' AND parent_id != '' AND parent_id NOT IN (SELECT id FROM files)")

	return &DB{Conn: conn}, nil
}


// Close closes the database connection.
func (db *DB) Close() error {
	return db.Conn.Close()
}

// SaveSetting saves a key-value setting.
func (db *DB) SaveSetting(key, value string) error {
	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
	return err
}

// GetSetting retrieves a setting value by key.
func (db *DB) GetSetting(key string) (string, error) {
	var val string
	err := db.Conn.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

// GetAccounts retrieves all accounts with decrypted credentials.
func (db *DB) GetAccounts() ([]AccountRecord, error) {
	rows, err := db.Conn.Query("SELECT id, provider, display_name, email, access_token, refresh_token, token_expiry, used_space, total_space, active FROM accounts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []AccountRecord
	for rows.Next() {
		var a AccountRecord
		var activeInt int
		err := rows.Scan(&a.ID, &a.Provider, &a.DisplayName, &a.Email, &a.AccessToken, &a.RefreshToken, &a.TokenExpiry, &a.UsedSpace, &a.TotalSpace, &activeInt)
		if err != nil {
			return nil, err
		}
		a.Active = activeInt == 1

		// Decrypt sensitive credential fields
		if decrypted, err := security.Decrypt(a.AccessToken); err == nil {
			a.AccessToken = decrypted
		}
		if decrypted, err := security.Decrypt(a.RefreshToken); err == nil {
			a.RefreshToken = decrypted
		}

		list = append(list, a)
	}
	return list, nil
}

// MigrateEncryptTokens re-saves all existing accounts to encrypt their tokens.
// This is safe to call multiple times — tokens already encrypted will be handled
// gracefully by the DPAPI fallback in Decrypt.
func (db *DB) MigrateEncryptTokens() {
	rows, err := db.Conn.Query("SELECT id, provider, display_name, email, access_token, refresh_token, token_expiry, used_space, total_space, active FROM accounts")
	if err != nil {
		log.Printf("MigrateEncryptTokens: failed to query accounts: %v", err)
		return
	}

	var accounts []AccountRecord
	for rows.Next() {
		var a AccountRecord
		var activeInt int
		if err := rows.Scan(&a.ID, &a.Provider, &a.DisplayName, &a.Email, &a.AccessToken, &a.RefreshToken, &a.TokenExpiry, &a.UsedSpace, &a.TotalSpace, &activeInt); err == nil {
			a.Active = activeInt == 1
			// Decrypt first to get plain-text (handles both plain and already-encrypted)
			if dec, err := security.Decrypt(a.AccessToken); err == nil {
				a.AccessToken = dec
			}
			if dec, err := security.Decrypt(a.RefreshToken); err == nil {
				a.RefreshToken = dec
			}
			accounts = append(accounts, a)
		}
	}
	rows.Close()

	for _, a := range accounts {
		if err := db.SaveAccount(a); err != nil {
			log.Printf("MigrateEncryptTokens: failed to re-save account %s: %v", a.ID, err)
		} else {
			log.Printf("MigrateEncryptTokens: encrypted credentials for account %s (%s)", a.DisplayName, a.Provider)
		}
	}
}

// SaveAccount saves or updates an account.
func (db *DB) SaveAccount(a AccountRecord) error {
	activeInt := 0
	if a.Active {
		activeInt = 1
	}

	// Encrypt sensitive credential fields before storing
	encAccessToken, err := security.Encrypt(a.AccessToken)
	if err != nil {
		log.Printf("Warning: failed to encrypt access_token for account %s: %v", a.ID, err)
		encAccessToken = a.AccessToken // fallback to plain-text
	}
	encRefreshToken, err := security.Encrypt(a.RefreshToken)
	if err != nil {
		log.Printf("Warning: failed to encrypt refresh_token for account %s: %v", a.ID, err)
		encRefreshToken = a.RefreshToken
	}

	_, err = db.Exec(
		"INSERT OR REPLACE INTO accounts (id, provider, display_name, email, access_token, refresh_token, token_expiry, used_space, total_space, active) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		a.ID, a.Provider, a.DisplayName, a.Email, encAccessToken, encRefreshToken, a.TokenExpiry, a.UsedSpace, a.TotalSpace, activeInt,
	)
	return err
}

// DeleteAccount deletes an account and all its files from local index.
func (db *DB) DeleteAccount(id string) error {
	_, err := db.Exec("DELETE FROM accounts WHERE id = ?", id)
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM files WHERE account_id = ?", id)
	return err
}

// SaveFile saves or updates a file metadata record.
func (db *DB) SaveFile(f FileRecord) error {
	starredInt := 0
	if f.Starred {
		starredInt = 1
	}
	isFolderInt := 0
	if f.IsFolder {
		isFolderInt = 1
	}
	sharedInt := 0
	if f.Shared {
		sharedInt = 1
	}
	_, err := db.Exec(
		"INSERT OR REPLACE INTO files (id, name, size, is_folder, parent_id, provider, account_id, physical_id, created_at, modified_at, starred, shared) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		f.ID, f.Name, f.Size, isFolderInt, f.ParentID, f.Provider, f.AccountID, f.PhysicalID, f.CreatedAt.Format(time.RFC3339), f.ModifiedAt.Format(time.RFC3339), starredInt, sharedInt,
	)
	return err
}

// GetFiles retrieves files matching query parameters.
func (db *DB) GetFiles(parentID string, starredOnly bool, searchKeyword string) ([]FileRecord, error) {
	query := "SELECT id, name, size, is_folder, parent_id, provider, account_id, physical_id, created_at, modified_at, starred, shared FROM files WHERE deleted = 0"
	var args []interface{}

	if searchKeyword != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+searchKeyword+"%")
	} else {
		query += " AND parent_id = ?"
		args = append(args, parentID)
	}

	if starredOnly {
		query += " AND starred = 1"
	}

	// Order folders first, then alphabetically by name
	query += " ORDER BY is_folder DESC, name ASC"

	rows, err := db.Conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []FileRecord
	for rows.Next() {
		var f FileRecord
		var isFolderInt, starredInt, sharedInt int
		var createdStr, modifiedStr string

		err := rows.Scan(&f.ID, &f.Name, &f.Size, &isFolderInt, &f.ParentID, &f.Provider, &f.AccountID, &f.PhysicalID, &createdStr, &modifiedStr, &starredInt, &sharedInt)
		if err != nil {
			return nil, err
		}

		f.IsFolder = isFolderInt == 1
		f.Starred = starredInt == 1
		f.Shared = sharedInt == 1

		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			f.CreatedAt = t
		} else {
			f.CreatedAt = time.Now()
		}
		if t, err := time.Parse(time.RFC3339, modifiedStr); err == nil {
			f.ModifiedAt = t
		} else {
			f.ModifiedAt = time.Now()
		}

		list = append(list, f)
	}
	return list, nil
}

// GetFile retrieves a specific file.
func (db *DB) GetFile(id string) (FileRecord, error) {
	var f FileRecord
	var isFolderInt, starredInt, sharedInt int
	var createdStr, modifiedStr string

	err := db.Conn.QueryRow(
		"SELECT id, name, size, is_folder, parent_id, provider, account_id, physical_id, created_at, modified_at, starred, shared FROM files WHERE id = ?", id,
	).Scan(&f.ID, &f.Name, &f.Size, &isFolderInt, &f.ParentID, &f.Provider, &f.AccountID, &f.PhysicalID, &createdStr, &modifiedStr, &starredInt, &sharedInt)

	if err != nil {
		return f, err
	}

	f.IsFolder = isFolderInt == 1
	f.Starred = starredInt == 1
	f.Shared = sharedInt == 1

	if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
		f.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, modifiedStr); err == nil {
		f.ModifiedAt = t
	}

	return f, nil
}

func (db *DB) GetFileByNameAndParent(name string, parentID string) (FileRecord, error) {
	var f FileRecord
	var isFolderInt, starredInt, sharedInt int
	var createdStr, modifiedStr string

	err := db.Conn.QueryRow(
		"SELECT id, name, size, is_folder, parent_id, provider, account_id, physical_id, created_at, modified_at, starred, shared FROM files WHERE name = ? AND parent_id = ? AND deleted = 0",
		name, parentID,
	).Scan(&f.ID, &f.Name, &f.Size, &isFolderInt, &f.ParentID, &f.Provider, &f.AccountID, &f.PhysicalID, &createdStr, &modifiedStr, &starredInt, &sharedInt)

	if err != nil {
		return f, err
	}

	f.IsFolder = isFolderInt == 1
	f.Starred = starredInt == 1
	f.Shared = sharedInt == 1

	if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
		f.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, modifiedStr); err == nil {
		f.ModifiedAt = t
	}

	return f, nil
}

func (db *DB) GetFileByName(name string) (FileRecord, error) {
	var f FileRecord
	var isFolderInt, starredInt, sharedInt int
	var createdStr, modifiedStr string

	err := db.Conn.QueryRow(
		"SELECT id, name, size, is_folder, parent_id, provider, account_id, physical_id, created_at, modified_at, starred, shared FROM files WHERE name = ? AND deleted = 0 LIMIT 1",
		name,
	).Scan(&f.ID, &f.Name, &f.Size, &isFolderInt, &f.ParentID, &f.Provider, &f.AccountID, &f.PhysicalID, &createdStr, &modifiedStr, &starredInt, &sharedInt)

	if err != nil {
		return f, err
	}

	f.IsFolder = isFolderInt == 1
	f.Starred = starredInt == 1
	f.Shared = sharedInt == 1

	if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
		f.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, modifiedStr); err == nil {
		f.ModifiedAt = t
	}

	return f, nil
}

// FindFileByPhysicalID searches for a file by its physical ID map pattern.
func (db *DB) FindFileByPhysicalID(accountID, physicalID string) (FileRecord, error) {
	var f FileRecord
	var isFolderInt, starredInt, sharedInt int
	var createdStr, modifiedStr string

	pattern := "%\"" + accountID + "\":\"" + physicalID + "\"%"
	err := db.Conn.QueryRow(
		"SELECT id, name, size, is_folder, parent_id, provider, account_id, physical_id, created_at, modified_at, starred, shared FROM files WHERE deleted = 0 AND physical_id LIKE ?", pattern,
	).Scan(&f.ID, &f.Name, &f.Size, &isFolderInt, &f.ParentID, &f.Provider, &f.AccountID, &f.PhysicalID, &createdStr, &modifiedStr, &starredInt, &sharedInt)

	if err != nil {
		return f, err
	}

	f.IsFolder = isFolderInt == 1
	f.Starred = starredInt == 1
	f.Shared = sharedInt == 1

	if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
		f.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, modifiedStr); err == nil {
		f.ModifiedAt = t
	}

	return f, nil
}

// GetFilesByAccount retrieves all active file records associated with a specific cloud account ID.
func (db *DB) GetFilesByAccount(accountID string) ([]FileRecord, error) {
	rows, err := db.Conn.Query(
		"SELECT id, name, size, is_folder, parent_id, provider, account_id, physical_id, created_at, modified_at, starred, shared FROM files WHERE deleted = 0 AND (account_id = ? OR physical_id LIKE ?)",
		accountID, "%\""+accountID+"\":%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []FileRecord
	for rows.Next() {
		var f FileRecord
		var isFolderInt, starredInt, sharedInt int
		var createdStr, modifiedStr string

		err := rows.Scan(&f.ID, &f.Name, &f.Size, &isFolderInt, &f.ParentID, &f.Provider, &f.AccountID, &f.PhysicalID, &createdStr, &modifiedStr, &starredInt, &sharedInt)
		if err != nil {
			return nil, err
		}

		f.IsFolder = isFolderInt == 1
		f.Starred = starredInt == 1
		f.Shared = sharedInt == 1

		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			f.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, modifiedStr); err == nil {
			f.ModifiedAt = t
		}

		list = append(list, f)
	}
	return list, nil
}

// DeleteFile marks a file or folder record as deleted (soft delete), including its children recursively if it's a folder.
func (db *DB) DeleteFile(id string) error {
	// 1. Get all children if this is a folder
	rows, err := db.Conn.Query("SELECT id FROM files WHERE parent_id = ?", id)
	if err == nil {
		var childIDs []string
		for rows.Next() {
			var cid string
			if err := rows.Scan(&cid); err == nil {
				childIDs = append(childIDs, cid)
			}
		}
		rows.Close()

		for _, cid := range childIDs {
			_ = db.DeleteFile(cid) // recursive call to soft delete children
		}
	}

	// 2. Soft delete the record itself
	_, err = db.Exec("UPDATE files SET deleted = 1 WHERE id = ?", id)
	return err
}

// RestoreFile restores a file or folder record locally (deleted = 0), and recursively restores its parents.
func (db *DB) RestoreFile(id string) error {
	// 1. Restore this item
	_, err := db.Exec("UPDATE files SET deleted = 0 WHERE id = ?", id)
	if err != nil {
		return err
	}

	// 2. Restore all its parents recursively so it's visible in the tree again
	var parentID string
	err = db.Conn.QueryRow("SELECT parent_id FROM files WHERE id = ?", id).Scan(&parentID)
	if err == nil && parentID != "" && parentID != "root" {
		_ = db.RestoreFile(parentID)
	}
	return nil
}

// PermanentlyDeleteFile physically deletes the record and all children from the database.
func (db *DB) PermanentlyDeleteFile(id string) error {
	// 1. Get all children if this is a folder
	rows, err := db.Conn.Query("SELECT id FROM files WHERE parent_id = ?", id)
	if err == nil {
		var childIDs []string
		for rows.Next() {
			var cid string
			if err := rows.Scan(&cid); err == nil {
				childIDs = append(childIDs, cid)
			}
		}
		rows.Close()

		for _, cid := range childIDs {
			_ = db.PermanentlyDeleteFile(cid)
		}
	}

	// 2. Delete the record physically
	_, err = db.Exec("DELETE FROM files WHERE id = ?", id)
	return err
}

// GetTrashedFiles returns all records currently in the trash (deleted = 1).
func (db *DB) GetTrashedFiles() ([]FileRecord, error) {
	query := "SELECT id, name, size, is_folder, parent_id, provider, account_id, physical_id, created_at, modified_at, starred, shared FROM files WHERE deleted = 1 ORDER BY is_folder DESC, name ASC"
	rows, err := db.Conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []FileRecord
	for rows.Next() {
		var f FileRecord
		var isFolderInt, starredInt, sharedInt int
		var createdStr, modifiedStr string

		err := rows.Scan(&f.ID, &f.Name, &f.Size, &isFolderInt, &f.ParentID, &f.Provider, &f.AccountID, &f.PhysicalID, &createdStr, &modifiedStr, &starredInt, &sharedInt)
		if err != nil {
			return nil, err
		}

		f.IsFolder = isFolderInt == 1
		f.Starred = starredInt == 1
		f.Shared = sharedInt == 1

		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			f.CreatedAt = t
		} else {
			f.CreatedAt = time.Now()
		}
		if t, err := time.Parse(time.RFC3339, modifiedStr); err == nil {
			f.ModifiedAt = t
		} else {
			f.ModifiedAt = time.Now()
		}

		list = append(list, f)
	}
	return list, nil
}

// UpdateStarred toggles a file's starred state.
func (db *DB) UpdateStarred(id string, starred bool) error {
	val := 0
	if starred {
		val = 1
	}
	_, err := db.Exec("UPDATE files SET starred = ? WHERE id = ?", val, id)
	return err
}

// GetRecentFiles retrieves the most recently modified files across all providers.
func (db *DB) GetRecentFiles(limit int) ([]FileRecord, error) {
	query := `SELECT id, name, size, is_folder, parent_id, provider, account_id, physical_id, created_at, modified_at, starred, shared 
	          FROM files 
	          WHERE deleted = 0 AND is_folder = 0 AND id != 'root'
	          ORDER BY datetime(modified_at) DESC LIMIT ?`
	rows, err := db.Conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []FileRecord
	for rows.Next() {
		var f FileRecord
		var isFolderInt, starredInt, sharedInt int
		var createdStr, modifiedStr string

		err := rows.Scan(&f.ID, &f.Name, &f.Size, &isFolderInt, &f.ParentID, &f.Provider, &f.AccountID, &f.PhysicalID, &createdStr, &modifiedStr, &starredInt, &sharedInt)
		if err != nil {
			return nil, err
		}

		f.IsFolder = isFolderInt == 1
		f.Starred = starredInt == 1
		f.Shared = sharedInt == 1

		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			f.CreatedAt = t
		} else {
			f.CreatedAt = time.Now()
		}
		if t, err := time.Parse(time.RFC3339, modifiedStr); err == nil {
			f.ModifiedAt = t
		} else {
			f.ModifiedAt = time.Now()
		}

		list = append(list, f)
	}
	return list, nil
}

// UpdateFileName renames a file record.
func (db *DB) UpdateFileName(id string, name string) error {
	_, err := db.Exec("UPDATE files SET name = ?, modified_at = ? WHERE id = ?", name, time.Now().Format(time.RFC3339), id)
	return err
}

type ActivityRecord struct {
	ID        int64  `json:"id"`
	FileID    string `json:"fileId"`
	FileName  string `json:"fileName"`
	Action    string `json:"action"`
	Details   string `json:"details"`
	Timestamp string `json:"timestamp"`
}

func (db *DB) LogActivity(fileID, fileName, action, details string) error {
	_, err := db.Exec(
		"INSERT INTO activities (file_id, file_name, action, details, timestamp) VALUES (?, ?, ?, ?, ?)",
		fileID, fileName, action, details, time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	// Prune activity log to keep only the latest 1000 entries
	_, _ = db.Exec("DELETE FROM activities WHERE id NOT IN (SELECT id FROM activities ORDER BY id DESC LIMIT 1000)")
	return nil
}

func (db *DB) GetFileActivities(fileID string) ([]ActivityRecord, error) {
	rows, err := db.Conn.Query(
		"SELECT id, file_id, file_name, action, details, timestamp FROM activities WHERE file_id = ? ORDER BY id DESC",
		fileID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ActivityRecord
	for rows.Next() {
		var r ActivityRecord
		if err := rows.Scan(&r.ID, &r.FileID, &r.FileName, &r.Action, &r.Details, &r.Timestamp); err == nil {
			list = append(list, r)
		}
	}
	return list, nil
}

func (db *DB) GetGeneralActivities(limit int) ([]ActivityRecord, error) {
	rows, err := db.Conn.Query(
		"SELECT id, file_id, file_name, action, details, timestamp FROM activities ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ActivityRecord
	for rows.Next() {
		var r ActivityRecord
		if err := rows.Scan(&r.ID, &r.FileID, &r.FileName, &r.Action, &r.Details, &r.Timestamp); err == nil {
			list = append(list, r)
		}
	}
	return list, nil
}

func (db *DB) GetSyncTasks() ([]SyncTask, error) {
	rows, err := db.Conn.Query("SELECT id, local_path, target_folder_id, account_id, sync_mode, enabled, last_sync FROM sync_tasks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []SyncTask
	for rows.Next() {
		var t SyncTask
		var enabledVal int
		var lastSync sql.NullString
		if err := rows.Scan(&t.ID, &t.LocalPath, &t.TargetFolderID, &t.AccountID, &t.SyncMode, &enabledVal, &lastSync); err == nil {
			t.Enabled = enabledVal == 1
			if lastSync.Valid {
				t.LastSync = lastSync.String
			}
			list = append(list, t)
		}
	}
	return list, nil
}

func (db *DB) GetSyncTaskByID(id string) (SyncTask, error) {
	var t SyncTask
	var enabledVal int
	var lastSync sql.NullString
	err := db.Conn.QueryRow("SELECT id, local_path, target_folder_id, account_id, sync_mode, enabled, last_sync FROM sync_tasks WHERE id = ?", id).Scan(&t.ID, &t.LocalPath, &t.TargetFolderID, &t.AccountID, &t.SyncMode, &enabledVal, &lastSync)
	if err != nil {
		return t, err
	}
	t.Enabled = enabledVal == 1
	if lastSync.Valid {
		t.LastSync = lastSync.String
	}
	return t, nil
}


func (db *DB) AddSyncTask(t SyncTask) error {
	enabledVal := 0
	if t.Enabled {
		enabledVal = 1
	}
	_, err := db.Exec(
		"INSERT INTO sync_tasks (id, local_path, target_folder_id, account_id, sync_mode, enabled, last_sync) VALUES (?, ?, ?, ?, ?, ?, ?)",
		t.ID, t.LocalPath, t.TargetFolderID, t.AccountID, t.SyncMode, enabledVal, t.LastSync,
	)
	return err
}

func (db *DB) DeleteSyncTask(id string) error {
	_, err := db.Exec("DELETE FROM sync_tasks WHERE id = ?", id)
	return err
}

func (db *DB) ToggleSyncTask(id string, enabled bool) error {
	enabledVal := 0
	if enabled {
		enabledVal = 1
	}
	_, err := db.Exec("UPDATE sync_tasks SET enabled = ? WHERE id = ?", enabledVal, id)
	return err
}

func (db *DB) UpdateSyncTaskLastSync(id string, lastSync string) error {
	_, err := db.Exec("UPDATE sync_tasks SET last_sync = ? WHERE id = ?", lastSync, id)
	return err
}

func (db *DB) UpdateSyncTask(id string, targetFolderID string, accountID string, syncMode string) error {
	_, err := db.Exec(
		"UPDATE sync_tasks SET target_folder_id = ?, account_id = ?, sync_mode = ? WHERE id = ?",
		targetFolderID, accountID, syncMode, id,
	)
	return err
}

