package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const uploadWebSocketAddr = "127.0.0.1:5999"
const uploadWebSocketPath = "/ws/uploads"

type progressMessage struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Filename   string `json:"filename"`
	BytesSent  int64  `json:"bytesSent,omitempty"`
	TotalBytes int64  `json:"totalBytes,omitempty"`
	Percent    int    `json:"percent,omitempty"`
	Error      string `json:"error,omitempty"`
}

type transferHub struct {
	mu        sync.Mutex
	clients   map[*websocket.Conn]struct{}
	server    *http.Server
	authToken string // Random UUID token required for WebSocket connections
}

func newTransferHub() *transferHub {
	return &transferHub{
		clients:   make(map[*websocket.Conn]struct{}),
		authToken: uuid.New().String(),
	}
}

func (h *transferHub) start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(uploadWebSocketPath, h.handleWebSocket)
	h.server = &http.Server{Addr: uploadWebSocketAddr, Handler: mux}
	listener, err := net.Listen("tcp", h.server.Addr)
	if err != nil {
		return err
	}

	go func() {
		if err := h.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("transfer websocket server error: %v", err)
		}
	}()

	return nil
}

func (h *transferHub) stop() {
	if h == nil || h.server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = h.server.Shutdown(ctx)

	h.mu.Lock()
	for conn := range h.clients {
		_ = conn.Close()
		delete(h.clients, conn)
	}
	h.mu.Unlock()
}

func (h *transferHub) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Validate auth token — accept via Authorization header or ?token= query param
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("X-WS-Token")
	}
	if token != h.authToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()

	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			_ = conn.Close()
		}()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func (h *transferHub) broadcastProgress(message progressMessage) {
	if h == nil {
		return
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			_ = conn.Close()
			delete(h.clients, conn)
		}
	}
}

type progressTracker struct {
	hub        *transferHub
	id         string
	filename   string
	totalBytes int64
	msgType    string

	mu        sync.Mutex
	sentBytes int64
	lastEmit  time.Time
}

func newProgressTracker(hub *transferHub, id, filename string, totalBytes int64, msgType string) *progressTracker {
	return &progressTracker{
		hub:        hub,
		id:         id,
		filename:   filename,
		totalBytes: totalBytes,
		msgType:    msgType,
		lastEmit:   time.Now(),
	}
}

func (t *progressTracker) reader(r io.Reader) io.Reader {
	return &progressReader{tracker: t, reader: r}
}

func (t *progressTracker) writer(w io.Writer) io.Writer {
	return &progressWriter{tracker: t, writer: w}
}

func (t *progressTracker) add(delta int64, force bool) {
	if t == nil || t.hub == nil {
		return
	}

	t.mu.Lock()
	t.sentBytes += delta
	sentBytes := t.sentBytes
	totalBytes := t.totalBytes
	shouldEmit := force || time.Since(t.lastEmit) >= 120*time.Millisecond || (totalBytes > 0 && sentBytes >= totalBytes)
	if shouldEmit {
		t.lastEmit = time.Now()
	}
	t.mu.Unlock()

	if !shouldEmit {
		return
	}

	percent := 0
	if totalBytes > 0 {
		percent = int((sentBytes * 100) / totalBytes)
		if percent > 100 {
			percent = 100
		}
	}

	t.hub.broadcastProgress(progressMessage{
		Type:       t.msgType,
		ID:         t.id,
		Filename:   t.filename,
		BytesSent:  sentBytes,
		TotalBytes: totalBytes,
		Percent:    percent,
	})
}

type progressReader struct {
	tracker *progressTracker
	reader  io.Reader
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.tracker.add(int64(n), false)
	}
	if err == io.EOF {
		r.tracker.add(0, true)
	}
	return n, err
}

type progressWriter struct {
	tracker *progressTracker
	writer  io.Writer
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if n > 0 {
		w.tracker.add(int64(n), false)
	}
	return n, err
}

func (a *App) startUploadWebSocketServer() {
	if a.uploadHub != nil {
		return
	}

	hub := newTransferHub()
	if err := hub.start(); err != nil {
		log.Printf("failed to start transfer websocket server: %v", err)
		return
	}

	a.uploadHub = hub
	log.Printf("transfer websocket server listening on ws://%s%s", uploadWebSocketAddr, uploadWebSocketPath)
}

func (a *App) stopUploadWebSocketServer() {
	if a.uploadHub == nil {
		return
	}

	a.uploadHub.stop()
	a.uploadHub = nil
}
