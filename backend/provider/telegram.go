package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// TelegramProvider implements the Provider interface for Telegram Channel Storage.
// It maps credentials as follows:
// - Email: Bot Token
// - AccessToken: Channel/Chat ID (e.g. -10012345678)
// - RefreshToken: "telegram"
// - DisplayName: Custom User Account Label
type TelegramProvider struct {
	BotToken string
	ChatID   string
	client   *http.Client
}

func NewTelegramProvider(botToken, chatID string) *TelegramProvider {
	return &TelegramProvider{
		BotToken: botToken,
		ChatID:   chatID,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

type tgMeResponse struct {
	Ok     bool `json:"ok"`
	Result struct {
		FirstName string `json:"first_name"`
		Username  string `json:"username"`
	} `json:"result"`
}

func (t *TelegramProvider) GetUserInfo() (name, email string, err error) {
	u := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", t.BotToken)
	resp, err := t.client.Get(u)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("invalid bot token (status %d)", resp.StatusCode)
	}

	var me tgMeResponse
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return "", "", err
	}

	return me.Result.FirstName, "@" + me.Result.Username, nil
}

func (t *TelegramProvider) GetQuota() (used, total int64, err error) {
	// Telegram storage is unlimited, return placeholder values (0 used, 100 TB total)
	return 0, 100 * 1024 * 1024 * 1024 * 1024, nil
}

func (t *TelegramProvider) ListDirectory(remoteParentID string) ([]FileMetadata, error) {
	// Telegram is flat; all directory traversal is virtual and handled by our SQLite virtual db.
	return nil, nil
}

func (t *TelegramProvider) CreateFolder(remoteParentID string, name string) (string, error) {
	// Virtual directories; nothing to create on Telegram.
	return "virtual", nil
}

func (t *TelegramProvider) UploadFile(remoteParentID string, filename string, r io.Reader, size int64) (string, error) {
	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)

	err := writer.WriteField("chat_id", t.ChatID)
	if err != nil {
		return "", err
	}

	part, err := writer.CreateFormFile("document", filename)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(part, r)
	if err != nil {
		return "", err
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}

	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", t.BotToken)
	req, err := http.NewRequest("POST", u, bodyBuf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("telegram sendDocument failed: %s", string(respBody))
	}

	var res struct {
		Ok     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
			Document  struct {
				FileID string `json:"file_id"`
			} `json:"document"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	// We serialize physicalID as "message_id|file_id"
	physicalID := fmt.Sprintf("%d|%s", res.Result.MessageID, res.Result.Document.FileID)
	return physicalID, nil
}

func (t *TelegramProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	parts := strings.Split(physicalID, "|")
	fileID := physicalID
	if len(parts) > 1 {
		fileID = parts[1]
	}

	// 1. Get file path
	u := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", t.BotToken, fileID)
	resp, err := t.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("telegram getFile failed: %s", string(respBody))
	}

	var res struct {
		Ok     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	// 2. Stream file content from CDN
	downloadURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", t.BotToken, res.Result.FilePath)
	contentResp, err := t.client.Get(downloadURL)
	if err != nil {
		return nil, err
	}

	if contentResp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, contentResp.Body)
		contentResp.Body.Close()
		return nil, fmt.Errorf("failed to download from telegram CDN: status %d", contentResp.StatusCode)
	}

	return contentResp.Body, nil
}

func (t *TelegramProvider) DeleteFile(physicalID string) error {
	parts := strings.Split(physicalID, "|")
	if len(parts) < 2 {
		return nil // no message ID to delete
	}
	messageID := parts[0]

	u := fmt.Sprintf("https://api.telegram.org/bot%s/deleteMessage?chat_id=%s&message_id=%s", t.BotToken, t.ChatID, messageID)
	resp, err := t.client.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (t *TelegramProvider) RenameFile(physicalID string, newName string) error {
	// Directory cataloging is virtual; nothing to rename physically on Telegram.
	return nil
}

func (t *TelegramProvider) GetOAuthURL() string {
	return ""
}
