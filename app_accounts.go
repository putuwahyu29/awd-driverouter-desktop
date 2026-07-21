package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"driverouter/backend/db"
	"driverouter/backend/provider"
	"driverouter/backend/sync"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/oauth2"
)

// GetAccounts lists connected accounts.
func (a *App) GetAccounts() ([]db.AccountRecord, error) {
	return a.database.GetAccounts()
}

// DisconnectAccount unlinks an account.
func (a *App) DisconnectAccount(accountID string) error {
	return a.database.DeleteAccount(accountID)
}

// GetSettings returns application settings.
func (a *App) GetSettings() (map[string]string, error) {
	strategy, err := a.database.GetSetting("upload_strategy")
	if err != nil {
		strategy = "round_robin"
	}

	googleClientID, _ := a.database.GetSetting("google_client_id")
	googleClientSecret, _ := a.database.GetSetting("google_client_secret")
	onedriveClientID, _ := a.database.GetSetting("onedrive_client_id")
	onedriveClientSecret, _ := a.database.GetSetting("onedrive_client_secret")
	dropboxClientID, _ := a.database.GetSetting("dropbox_client_id")
	dropboxClientSecret, _ := a.database.GetSetting("dropbox_client_secret")
	boxClientID, _ := a.database.GetSetting("box_client_id")
	boxClientSecret, _ := a.database.GetSetting("box_client_secret")
	yandexClientID, _ := a.database.GetSetting("yandex_client_id")
	yandexClientSecret, _ := a.database.GetSetting("yandex_client_secret")
	pcloudClientID, _ := a.database.GetSetting("pcloud_client_id")
	pcloudClientSecret, _ := a.database.GetSetting("pcloud_client_secret")
	telegramAPIID, _ := a.database.GetSetting("telegram_api_id")
	telegramAPIHash, _ := a.database.GetSetting("telegram_api_hash")
	language, _ := a.database.GetSetting("language")
	if language == "" {
		language = "en"
	}
	minimizeToTray, _ := a.database.GetSetting("minimize_to_tray")
	if minimizeToTray == "" {
		minimizeToTray = "true"
	}
	backupInterval, _ := a.database.GetSetting("backup_interval")
	if backupInterval == "" {
		backupInterval = "60"
	}

	return map[string]string{
		"upload_strategy":        strategy,
		"google_client_id":       googleClientID,
		"google_client_secret":   googleClientSecret,
		"onedrive_client_id":     onedriveClientID,
		"onedrive_client_secret": onedriveClientSecret,
		"dropbox_client_id":      dropboxClientID,
		"dropbox_client_secret":  dropboxClientSecret,
		"box_client_id":          boxClientID,
		"box_client_secret":      boxClientSecret,
		"yandex_client_id":       yandexClientID,
		"yandex_client_secret":   yandexClientSecret,
		"pcloud_client_id":       pcloudClientID,
		"pcloud_client_secret":   pcloudClientSecret,
		"telegram_api_id":        telegramAPIID,
		"telegram_api_hash":      telegramAPIHash,
		"language":               language,
		"minimize_to_tray":       minimizeToTray,
		"backup_interval":        backupInterval,
	}, nil
}

// SaveSetting saves a single setting.
func (a *App) SaveSetting(key, value string) error {
	return a.database.SaveSetting(key, value)
}

// SaveCredentials saves user-supplied OAuth client credentials.
func (a *App) SaveCredentials(providerName, clientID, clientSecret string) error {
	cidKey := providerName + "_client_id"
	secretKey := providerName + "_client_secret"
	if providerName == "telegram_user" {
		cidKey = "telegram_api_id"
		secretKey = "telegram_api_hash"
	}
	if err := a.database.SaveSetting(cidKey, clientID); err != nil {
		return err
	}
	return a.database.SaveSetting(secretKey, clientSecret)
}

// StartOAuthFlow starts OAuth flow, opens browser, listens for redirect, saves account info.
func (a *App) StartOAuthFlow(providerName string) (*db.AccountRecord, error) {
	// Create state token
	state := uuid.New().String()

	// Get config credentials, fallback to defaults
	clientID, clientSecret := a.getOAuthCredentials(providerName)
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("client ID or Secret is empty. please configure credentials in settings first")
	}

	var authURL string
	var config *oauth2.Config

	switch providerName {
	case "google":
		config = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
			RedirectURL: "http://localhost:5998/oauth/callback",
			Scopes:      []string{"https://www.googleapis.com/auth/drive", "https://www.googleapis.com/auth/userinfo.profile", "https://www.googleapis.com/auth/userinfo.email"},
		}
		// Request offline access to get a Refresh Token
		authURL = config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	case "onedrive":
		config = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
				TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			},
			RedirectURL: "http://localhost:5998/oauth/callback",
			Scopes:      []string{"files.readwrite", "offline_access", "User.Read"},
		}
		authURL = config.AuthCodeURL(state)

	case "dropbox":
		config = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.dropbox.com/oauth2/authorize",
				TokenURL: "https://api.dropboxapi.com/oauth2/token",
			},
			RedirectURL: "http://localhost:5998/oauth/callback",
		}
		// Dropbox specific code for offline token type
		authURL = config.AuthCodeURL(state) + "&token_access_type=offline"

	case "box":
		config = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://account.box.com/api/oauth2/authorize",
				TokenURL: "https://api.box.com/oauth2/token",
			},
			RedirectURL: "http://localhost:5998/oauth/callback",
		}
		authURL = config.AuthCodeURL(state)

	case "yandex":
		config = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://oauth.yandex.com/authorize",
				TokenURL: "https://oauth.yandex.com/token",
			},
			RedirectURL: "http://localhost:5998/oauth/callback",
		}
		authURL = config.AuthCodeURL(state)

	case "pcloud":
		config = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://my.pcloud.com/oauth2/authorize",
				TokenURL: "https://api.pcloud.com/oauth2_token",
			},
			RedirectURL: "http://localhost:5998/oauth/callback",
		}
		authURL = config.AuthCodeURL(state)

	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", providerName)
	}

	// Start local listener in background
	go func() {
		time.Sleep(500 * time.Millisecond)
		runtime.BrowserOpenURL(a.ctx, authURL)
	}()

	code, err := provider.StartOAuthListener(state)
	if err != nil {
		if strings.Contains(err.Error(), "state mismatch") {
			return nil, fmt.Errorf("failed to complete authentication: this login session is outdated or was opened twice; close any old OAuth tab and try again: %w", err)
		}
		return nil, fmt.Errorf("failed to complete authentication: %w", err)
	}

	// Exchange authorization code for tokens
	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange auth code: %w", err)
	}

	// Create temporary provider client to fetch user details
	var p provider.Provider
	switch providerName {
	case "google":
		p = provider.NewGoogleDriveProvider(clientID, clientSecret, tok, nil)
	case "onedrive":
		p = provider.NewOneDriveProvider(clientID, clientSecret, tok, nil)
	case "dropbox":
		p = provider.NewDropboxProvider(clientID, clientSecret, tok, nil)
	case "box":
		p = provider.NewBoxProvider(clientID, clientSecret, tok, nil)
	case "yandex":
		p = provider.NewYandexProvider(clientID, clientSecret, tok, nil)
	case "pcloud":
		p = provider.NewPCloudProvider(clientID, clientSecret, tok, nil)
	}

	name, email, err := p.GetUserInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account user info: %w", err)
	}

	used, total, err := p.GetQuota()
	if err != nil {
		log.Printf("Warning: failed to retrieve drive quota: %v", err)
	}

	acc := db.AccountRecord{
		ID:           uuid.New().String(),
		Provider:     providerName,
		DisplayName:  name,
		Email:        email,
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenExpiry:  tok.Expiry.Format(time.RFC3339),
		UsedSpace:    used,
		TotalSpace:   total,
		Active:       true,
	}

	err = a.database.SaveAccount(acc)
	if err != nil {
		return nil, fmt.Errorf("failed to save account to db: %w", err)
	}

	// Cache the initial token in-memory
	sync.CacheToken(acc.ID, tok)

	// Trigger sync in background for this new account
	go func() {
		_ = a.syncMgr.SyncAccount(acc, p)
	}()

	return &acc, nil
}

// AddWebDAVAccount verifies credentials and adds a manual WebDAV account
func (a *App) AddWebDAVAccount(displayName, serverURL, username, password string) (*db.AccountRecord, error) {
	p := provider.NewWebDAVProvider(username, password, serverURL)

	// Verify connection and retrieve initial quota
	used, total, err := p.GetQuota()
	if err != nil {
		return nil, fmt.Errorf("failed to verify WebDAV credentials: %w", err)
	}

	acc := db.AccountRecord{
		ID:           uuid.New().String(),
		Provider:     "webdav",
		DisplayName:  displayName,
		Email:        username,
		AccessToken:  password,
		RefreshToken: serverURL,
		TokenExpiry:  time.Now().AddDate(100, 0, 0).Format(time.RFC3339),
		UsedSpace:    used,
		TotalSpace:   total,
		Active:       true,
	}

	err = a.database.SaveAccount(acc)
	if err != nil {
		return nil, fmt.Errorf("failed to save WebDAV account to database: %w", err)
	}

	// Trigger async sync crawling in background
	go func() {
		_ = a.syncMgr.SyncAccount(acc, p)
	}()

	return &acc, nil
}

// AddS3Account verifies credentials and adds a manual S3 account.
func (a *App) AddS3Account(displayName, serverURL, bucketName, accessKey, secretKey string) (*db.AccountRecord, error) {
	p := provider.NewS3Provider(accessKey, secretKey, serverURL, bucketName)

	// Verify connection by checking quota (this tests S3 credentials and endpoint)
	_, _, err := p.GetQuota()
	if err != nil {
		return nil, fmt.Errorf("failed to verify S3 credentials: %w", err)
	}

	acc := db.AccountRecord{
		ID:           uuid.New().String(),
		Provider:     "s3",
		DisplayName:  displayName,
		Email:        accessKey,
		AccessToken:  secretKey,
		RefreshToken: serverURL,
		TokenExpiry:  bucketName,
		UsedSpace:    0,
		TotalSpace:   100 * 1024 * 1024 * 1024 * 1024, // 100 TB placeholder
		Active:       true,
	}

	err = a.database.SaveAccount(acc)
	if err != nil {
		return nil, fmt.Errorf("failed to save S3 account to database: %w", err)
	}

	// Trigger async sync crawling in background
	go func() {
		_ = a.syncMgr.SyncAccount(acc, p)
	}()

	return &acc, nil
}

// AddTelegramAccount verifies credentials and adds a Telegram Bot Account.
func (a *App) AddTelegramAccount(displayName, botToken, chatID string) (*db.AccountRecord, error) {
	p := provider.NewTelegramProvider(botToken, chatID)

	// Verify connection by calling getMe
	botName, _, err := p.GetUserInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to verify Telegram Bot Token: %w", err)
	}

	acc := db.AccountRecord{
		ID:           uuid.New().String(),
		Provider:     "telegram",
		DisplayName:  displayName + " (" + botName + ")",
		Email:        botToken,
		AccessToken:  chatID,
		RefreshToken: "telegram",
		TokenExpiry:  time.Now().AddDate(100, 0, 0).Format(time.RFC3339),
		UsedSpace:    0,
		TotalSpace:   100 * 1024 * 1024 * 1024 * 1024, // 100 TB placeholder
		Active:       true,
	}

	err = a.database.SaveAccount(acc)
	if err != nil {
		return nil, fmt.Errorf("failed to save Telegram account to database: %w", err)
	}

	// Trigger async sync crawling in background
	go func() {
		_ = a.syncMgr.SyncAccount(acc, p)
	}()

	return &acc, nil
}

// SendTelegramCode sends a login verification code to the phone number via MTProto.
func (a *App) SendTelegramCode(phone string) (string, error) {
	apiIDStr, _ := a.database.GetSetting("telegram_api_id")
	apiHash, _ := a.database.GetSetting("telegram_api_hash")
	if apiIDStr == "" || apiHash == "" {
		return "", fmt.Errorf("Telegram API ID and API Hash must be configured in Settings first")
	}
	apiID := 0
	fmt.Sscanf(apiIDStr, "%d", &apiID)
	if apiID == 0 {
		return "", fmt.Errorf("Invalid Telegram API ID")
	}

	hash, err := provider.GetLoginHelper().StartLogin(phone, apiID, apiHash)
	if err != nil {
		return "", fmt.Errorf("failed to send Telegram code: %w", err)
	}
	return hash, nil
}

// VerifyTelegramCode completes the Telegram MTProto login and registers the account.
func (a *App) VerifyTelegramCode(code, password, displayName string) (*db.AccountRecord, error) {
	sessionData, err := provider.GetLoginHelper().VerifyCode(code, password)
	if err != nil {
		return nil, err
	}

	if sessionData == "PASSWORD_REQUIRED" {
		return nil, fmt.Errorf("PASSWORD_REQUIRED")
	}

	helper := provider.GetLoginHelper()
	phone := helper.Phone
	apiID := helper.APIID
	apiHash := helper.APIHash

	// Create and save account record
	p := provider.NewTelegramUserProvider(phone, sessionData, apiID, apiHash, nil)

	name, email, err := p.GetUserInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	acc := db.AccountRecord{
		ID:           uuid.New().String(),
		Provider:     "telegram_user",
		DisplayName:  displayName + " (" + name + ")",
		Email:        email,
		AccessToken:  sessionData,
		RefreshToken: "telegram_user",
		TokenExpiry:  time.Now().AddDate(100, 0, 0).Format(time.RFC3339),
		UsedSpace:    0,
		TotalSpace:   100 * 1024 * 1024 * 1024 * 1024, // 100 TB placeholder
		Active:       true,
	}

	err = a.database.SaveAccount(acc)
	if err != nil {
		return nil, fmt.Errorf("failed to save Telegram User account to database: %w", err)
	}

	// Trigger async sync crawling in background
	go func() {
		_ = a.syncMgr.SyncAccount(acc, p)
	}()

	return &acc, nil
}

func (a *App) getAccount(id string) (db.AccountRecord, error) {
	accounts, err := a.database.GetAccounts()
	if err != nil {
		return db.AccountRecord{}, err
	}
	for _, acc := range accounts {
		if acc.ID == id {
			return acc, nil
		}
	}
	return db.AccountRecord{}, fmt.Errorf("account not found")
}

func (a *App) getOAuthCredentials(providerName string) (string, string) {
	clientID, _ := a.database.GetSetting(providerName + "_client_id")
	clientSecret, _ := a.database.GetSetting(providerName + "_client_secret")
	return clientID, clientSecret
}
