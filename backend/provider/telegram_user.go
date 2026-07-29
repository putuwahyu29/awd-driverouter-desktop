package provider

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
)

type TelegramCaptionMetadata struct {
	VFolderID    string
	GroupID      string
	PartIndex    int
	TotalParts   int
	OriginalName string
	IsFolder     bool
}

func parseTelegramCaptionMetadata(caption string) *TelegramCaptionMetadata {
	if !strings.HasPrefix(caption, "[TD_") || !strings.HasSuffix(caption, "]") {
		return nil
	}
	content := caption[1 : len(caption)-1]
	parts := strings.Split(content, "|")
	if len(parts) < 2 {
		return nil
	}

	meta := &TelegramCaptionMetadata{}
	for _, p := range parts[1:] {
		if strings.HasPrefix(p, "ID:") {
			meta.GroupID = strings.TrimPrefix(p, "ID:")
		} else if strings.HasPrefix(p, "VDIR:") {
			meta.VFolderID = strings.TrimPrefix(p, "VDIR:")
		} else if strings.HasPrefix(p, "NAME:") {
			meta.OriginalName = strings.TrimPrefix(p, "NAME:")
		} else if strings.HasPrefix(p, "PART:") {
			partStr := strings.TrimPrefix(p, "PART:")
			subParts := strings.Split(partStr, "/")
			if len(subParts) == 2 {
				meta.PartIndex, _ = strconv.Atoi(subParts[0])
				meta.TotalParts, _ = strconv.Atoi(subParts[1])
			}
		} else if p == "FOLDER:true" {
			meta.IsFolder = true
		}
	}
	return meta
}



// MemorySessionStorage implements gotd session.Storage.
type MemorySessionStorage struct {
	mu   sync.RWMutex
	data []byte
}

func (m *MemorySessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.data) == 0 {
		return nil, session.ErrNotFound
	}
	return m.data, nil
}

func (m *MemorySessionStorage) StoreSession(ctx context.Context, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = data
	return nil
}

// TelegramUserProvider manages MTProto storage in Saved Messages.
// Kredensial dipetakan ke db:
// - Email: Nomor Telepon
// - AccessToken: Base64 Serialized Session Data
// - RefreshToken: "telegram_user"
type TelegramUserProvider struct {
	Phone       string
	SessionData string
	APIID       int
	APIHash     string
	onRefresh   func(string)
}

func NewTelegramUserProvider(phone, sessionData string, apiID int, apiHash string, onRefresh func(string)) *TelegramUserProvider {
	return &TelegramUserProvider{
		Phone:       phone,
		SessionData: sessionData,
		APIID:       apiID,
		APIHash:     apiHash,
		onRefresh:   onRefresh,
	}
}

func (p *TelegramUserProvider) GetUserInfo() (string, string, error) {
	name := "Telegram User"
	email := p.Phone

	err := p.runClient(func(ctx context.Context, api *tg.Client) error {
		user, err := api.UsersGetUsers(ctx, []tg.InputUserClass{&tg.InputUserSelf{}})
		if err != nil {
			return err
		}
		if len(user) > 0 {
			if u, ok := user[0].(*tg.User); ok {
				name = strings.TrimSpace(u.FirstName + " " + u.LastName)
				if u.Username != "" {
					email = "@" + u.Username
				}
			}
		}
		return nil
	})

	return name, email, err
}

func (p *TelegramUserProvider) GetQuota() (used, total int64, err error) {
	// Saved Messages has unlimited storage
	return 0, 100 * 1024 * 1024 * 1024 * 1024, nil
}

func parseTelegramPeer(remoteParentID string) (tg.InputPeerClass, error) {
	if remoteParentID == "" || remoteParentID == "root" || remoteParentID == "virtual" {
		return &tg.InputPeerSelf{}, nil
	}
	parts := strings.Split(remoteParentID, "|")
	if len(parts) < 2 {
		return &tg.InputPeerSelf{}, nil
	}
	var channelID, accessHash int64
	_, err1 := fmt.Sscan(parts[0], &channelID)
	_, err2 := fmt.Sscan(parts[1], &accessHash)
	if err1 != nil || err2 != nil {
		return &tg.InputPeerSelf{}, nil
	}
	return &tg.InputPeerChannel{
		ChannelID:  channelID,
		AccessHash: accessHash,
	}, nil
}

func (p *TelegramUserProvider) ListDirectory(remoteParentID string) ([]FileMetadata, error) {

	var items []FileMetadata

	peer, err := parseTelegramPeer(remoteParentID)
	if err != nil {
		return []FileMetadata{}, err
	}

	err = p.runClient(func(ctx context.Context, api *tg.Client) error {
		// Fetch Channels as Folders if at root level
		if remoteParentID == "" || remoteParentID == "root" || remoteParentID == "virtual" {
			dialogs, err := api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
				OffsetPeer: &tg.InputPeerEmpty{},
				Limit:      100,
			})
			if err == nil {
				if d, ok := dialogs.(interface{ GetChats() []tg.ChatClass }); ok {
					for _, chat := range d.GetChats() {
						if c, ok := chat.(*tg.Channel); ok {
							if !c.Megagroup && (c.Creator || c.AdminRights.EditMessages || c.AdminRights.DeleteMessages) {
								physID := fmt.Sprintf("%d|%d", c.ID, c.AccessHash)
								items = append(items, FileMetadata{
									ID:         physID,
									Name:       c.Title,
									Size:       0,
									IsFolder:   true,
									ParentID:   remoteParentID,
									Provider:   "telegram_user",
									PhysicalID: physID,
									CreatedAt:  time.Unix(int64(c.Date), 0),
									ModifiedAt: time.Unix(int64(c.Date), 0),
								})
							}
						}
					}
				}
			}
		}

		res, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:  peer,
			Limit: 100,
		})
		if err != nil {
			return err
		}

		if msgs, ok := res.(interface{ GetMessages() []tg.MessageClass }); ok {
			for _, m := range msgs.GetMessages() {
				if msg, ok := m.(*tg.Message); ok {
					if media, ok := msg.Media.(*tg.MessageMediaDocument); ok {
						if doc, ok := media.Document.(*tg.Document); ok {
							name := "document"
							for _, attr := range doc.Attributes {
								if filenameAttr, ok := attr.(*tg.DocumentAttributeFilename); ok {
									name = filenameAttr.FileName
								}
							}
							meta := parseTelegramCaptionMetadata(msg.Message)
							if meta != nil && meta.OriginalName != "" {
								name = meta.OriginalName
							}

							fileRefBase64 := base64.StdEncoding.EncodeToString(doc.FileReference)
							physID := fmt.Sprintf("%d|%d|%d|%s", msg.ID, doc.ID, doc.AccessHash, fileRefBase64)

							items = append(items, FileMetadata{
								ID:         physID,
								Name:       name,
								Size:       doc.Size,
								IsFolder:   meta != nil && meta.IsFolder,
								ParentID:   remoteParentID,
								Provider:   "telegram_user",
								PhysicalID: physID,
								CreatedAt:  time.Unix(int64(msg.Date), 0),
								ModifiedAt: time.Unix(int64(msg.Date), 0),
							})
						}
					}
				}
			}
		}
		return nil
	})

	return items, err
}

func (p *TelegramUserProvider) CreateFolder(remoteParentID string, name string) (string, error) {
	// Creating new folders (or backup tasks) creates Virtual Folders instead of channel spamming/FloodWait
	vFolderID := fmt.Sprintf("vfolder_%s_%d", uuid.New().String()[:8], time.Now().Unix())
	return vFolderID, nil
}

func (p *TelegramUserProvider) UploadFile(remoteParentID string, filename string, r io.Reader, size int64) (string, error) {
	var physicalID string

	peer, err := parseTelegramPeer(remoteParentID)
	if err != nil {
		return "", err
	}

	err = p.runClient(func(ctx context.Context, api *tg.Client) error {
		u := uploader.NewUploader(api).WithThreads(4)
		
		upload, err := u.FromReader(ctx, filename, r)
		if err != nil {
			return fmt.Errorf("failed to upload reader: %w", err)
		}

		caption := ""
		if strings.HasPrefix(remoteParentID, "vfolder_") || remoteParentID != "root" && remoteParentID != "" {
			caption = fmt.Sprintf("[TD_VDIR|VDIR:%s|NAME:%s]", remoteParentID, filename)
		}

		res, err := api.MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
			Peer:     peer,
			RandomID: rand.Int63(),
			Message:  caption,
			Media: &tg.InputMediaUploadedDocument{
				File:     upload,
				MimeType: "application/octet-stream",
				Attributes: []tg.DocumentAttributeClass{
					&tg.DocumentAttributeFilename{FileName: filename},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to send media: %w", err)
		}

		// Parse document metadata and message ID
		var doc *tg.Document
		var msgID int

		switch u := res.(type) {
		case *tg.Updates:
			for _, upd := range u.Updates {
				switch ev := upd.(type) {
				case *tg.UpdateNewMessage:
					if msg, ok := ev.Message.(*tg.Message); ok {
						msgID = msg.ID
						if media, ok := msg.Media.(*tg.MessageMediaDocument); ok {
							if d, ok := media.Document.(*tg.Document); ok {
								doc = d
							}
						}
					}
				case *tg.UpdateNewChannelMessage:
					if msg, ok := ev.Message.(*tg.Message); ok {
						msgID = msg.ID
						if media, ok := msg.Media.(*tg.MessageMediaDocument); ok {
							if d, ok := media.Document.(*tg.Document); ok {
								doc = d
							}
						}
					}
				}
			}
		}

		if doc == nil {
			return fmt.Errorf("failed to extract document metadata from telegram response")
		}

		fileRefBase64 := base64.StdEncoding.EncodeToString(doc.FileReference)
		physicalID = fmt.Sprintf("%d|%d|%d|%s", msgID, doc.ID, doc.AccessHash, fileRefBase64)
		return nil
	})

	return physicalID, err
}

func (p *TelegramUserProvider) DownloadFile(physicalID string) (io.ReadCloser, error) {
	parts := strings.Split(physicalID, "|")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid telegram_user physical ID structure")
	}

	docID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid doc ID: %w", err)
	}

	accessHash, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid access hash: %w", err)
	}

	fileRef, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return nil, fmt.Errorf("failed to decode file reference: %w", err)
	}

	pr, pw := io.Pipe()

	go func() {
		err := p.runClient(func(ctx context.Context, api *tg.Client) error {
			dl := downloader.NewDownloader()
			loc := &tg.InputDocumentFileLocation{
				ID:            docID,
				AccessHash:    accessHash,
				FileReference: fileRef,
			}

			_, err = dl.Download(api, loc).Stream(ctx, pw)
			return err
		})

		if err != nil {
			_ = pw.CloseWithError(err)
		} else {
			_ = pw.Close()
		}
	}()

	return pr, nil
}

func (p *TelegramUserProvider) DeleteFile(physicalID string) error {
	parts := strings.Split(physicalID, "|")
	if len(parts) == 0 {
		return fmt.Errorf("invalid physical ID")
	}

	msgID, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}

	return p.runClient(func(ctx context.Context, api *tg.Client) error {
		_, err := api.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
			Revoke: true,
			ID:     []int{msgID},
		})
		return err
	})
}

func (p *TelegramUserProvider) RenameFile(physicalID string, newName string) error {
	return nil
}

func (p *TelegramUserProvider) GetOAuthURL() string {
	return ""
}

// runClient runs a client session on-demand, executing the callback.
func (p *TelegramUserProvider) runClient(cb func(context.Context, *tg.Client) error) error {
	storage := &MemorySessionStorage{}
	if p.SessionData != "" {
		dec, err := base64.StdEncoding.DecodeString(p.SessionData)
		if err == nil {
			storage.data = dec
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client := telegram.NewClient(p.APIID, p.APIHash, telegram.Options{
		SessionStorage: storage,
	})

	var cbErr error
	err := client.Run(ctx, func(ctx context.Context) error {
		// Whenever session updates, bubble up to trigger callback
		go func() {
			for {
				time.Sleep(2 * time.Second)
				if ctx.Err() != nil {
					return
				}
				storage.mu.RLock()
				curData := storage.data
				storage.mu.RUnlock()
				if len(curData) > 0 {
					newEnc := base64.StdEncoding.EncodeToString(curData)
					if newEnc != p.SessionData {
						p.SessionData = newEnc
						if p.onRefresh != nil {
							p.onRefresh(newEnc)
						}
					}
				}
			}
		}()

		cbErr = cb(ctx, client.API())
		return nil
	})

	if err != nil {
		return err
	}
	return cbErr
}

// TelegramLoginHelper manages the state of active MTProto user logins.
type TelegramLoginHelper struct {
	client     *telegram.Client
	storage    *MemorySessionStorage
	Phone      string
	APIID      int
	APIHash    string
	authHash   string
	clientCtx  context.Context
	cancelFunc context.CancelFunc
	mu         sync.Mutex
}

var loginHelper *TelegramLoginHelper
var loginHelperMu sync.Mutex

func GetLoginHelper() *TelegramLoginHelper {
	loginHelperMu.Lock()
	defer loginHelperMu.Unlock()
	if loginHelper == nil {
		loginHelper = &TelegramLoginHelper{
			storage: &MemorySessionStorage{},
		}
	}
	return loginHelper
}

func (h *TelegramLoginHelper) StartLogin(phone string, apiID int, apiHash string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Phone = strings.TrimSpace(phone)
	h.APIID = apiID
	h.APIHash = strings.TrimSpace(apiHash)
	if h.cancelFunc != nil {
		h.cancelFunc()
	}

	// Always start with fresh session storage for a new login attempt
	h.storage = &MemorySessionStorage{}

	h.clientCtx, h.cancelFunc = context.WithCancel(context.Background())
	h.client = telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: h.storage,
	})

	// Start client loop in background
	go func() {
		_ = h.client.Run(h.clientCtx, func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		})
	}()

	// Wait for client connection
	time.Sleep(2 * time.Second)

	// Call SendCode
	ctx, cancel := context.WithTimeout(h.clientCtx, 15*time.Second)
	defer cancel()

	res, err := h.client.Auth().SendCode(ctx, h.Phone, auth.SendCodeOptions{})
	if err != nil {
		return "", err
	}

	sentCode, ok := res.(*tg.AuthSentCode)
	if !ok {
		return "", fmt.Errorf("unexpected telegram response structure")
	}

	h.authHash = sentCode.PhoneCodeHash
	return sentCode.PhoneCodeHash, nil
}

func (h *TelegramLoginHelper) VerifyCode(code string, password string) (sessionBase64 string, userName string, userEmail string, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.client == nil {
		return "", "", "", fmt.Errorf("login session not initialized")
	}

	ctx, cancel := context.WithTimeout(h.clientCtx, 20*time.Second)
	defer cancel()

	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	code = strings.ReplaceAll(code, "-", "")
	password = strings.TrimSpace(password)

	var signInErr error
	var authRes tg.AuthAuthorizationClass

	if password != "" {
		authRes, signInErr = h.client.Auth().Password(ctx, password)
	} else {
		authRes, signInErr = h.client.Auth().SignIn(ctx, h.Phone, code, h.authHash)
		if errors.Is(signInErr, auth.ErrPasswordAuthNeeded) || (signInErr != nil && strings.Contains(strings.ToLower(signInErr.Error()), "password")) {
			return "PASSWORD_REQUIRED", "", "", nil
		}
	}

	if signInErr != nil {
		return "", "", "", signInErr
	}

	uName := "Telegram User"
	uEmail := h.Phone
	if authRes != nil {
		if authResult, ok := authRes.(*tg.AuthAuthorization); ok {
			if u, ok := authResult.User.(*tg.User); ok {
				fullName := strings.TrimSpace(u.FirstName + " " + u.LastName)
				if fullName != "" {
					uName = fullName
				}
				if u.Username != "" {
					uEmail = "@" + u.Username
				}
			}
		}
	}

	// Capture the session data bytes
	h.storage.mu.RLock()
	sessionBytes := h.storage.data
	h.storage.mu.RUnlock()

	if len(sessionBytes) == 0 {
		return "", "", "", fmt.Errorf("failed to retrieve verified session data")
	}

	sessionBase64 = base64.StdEncoding.EncodeToString(sessionBytes)

	// Clean up login client
	if h.cancelFunc != nil {
		h.cancelFunc()
		h.client = nil
	}

	return sessionBase64, uName, uEmail, nil
}
