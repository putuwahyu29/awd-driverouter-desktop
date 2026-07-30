package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"
)

type HeadlessServer struct {
	app    *App
	host   string
	port   int
	apiKey string
}

func NewHeadlessServer(app *App, host string, port int, apiKey string) *HeadlessServer {
	return &HeadlessServer{
		app:    app,
		host:   host,
		port:   port,
		apiKey: apiKey,
	}
}

func (hs *HeadlessServer) Start() error {
	hs.app.isHeadless = true
	hs.app.headlessPort = hs.port

	// Initialize App backend if not already initialized
	if hs.app.database == nil {
		hs.app.startup(context.Background())
	}

	mux := http.NewServeMux()

	// Auth Middleware
	authWrapper := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if hs.apiKey != "" {
				clientKey := r.Header.Get("X-API-Key")
				if clientKey == "" {
					bearer := r.Header.Get("Authorization")
					if strings.HasPrefix(bearer, "Bearer ") {
						clientKey = strings.TrimPrefix(bearer, "Bearer ")
					}
				}
				if clientKey == "" {
					clientKey = r.URL.Query().Get("api_key")
				}
				if clientKey != hs.apiKey {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized: invalid or missing API Key"})
					return
				}
			}
			handler(w, r)
		}
	}

	// Health check endpoint
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"app":     "Awd DriveRouter",
			"mode":    "headless",
			"time":    time.Now().Unix(),
			"version": AppVersion,
		})
	})

	// Accounts API
	mux.HandleFunc("/api/GetAccounts", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		accounts, err := hs.app.GetAccounts()
		if err != nil {
			_ = json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		_ = json.NewEncoder(w).Encode(accounts)
	}))
	mux.HandleFunc("/api/accounts", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		accounts, err := hs.app.GetAccounts()
		if err != nil {
			_ = json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		_ = json.NewEncoder(w).Encode(accounts)
	}))

	// Files API
	mux.HandleFunc("/api/GetFiles", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		files, err := hs.app.GetFiles("root", false, "")
		if err != nil {
			_ = json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		_ = json.NewEncoder(w).Encode(files)
	}))

	// Settings API
	mux.HandleFunc("/api/GetSettings", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		settings, err := hs.app.GetSettings()
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]string{})
			return
		}
		_ = json.NewEncoder(w).Encode(settings)
	}))

	// Recent Files API
	mux.HandleFunc("/api/GetRecentFiles", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		files, err := hs.app.GetRecentFiles()
		if err != nil {
			_ = json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		_ = json.NewEncoder(w).Encode(files)
	}))

	// Starred Files API
	mux.HandleFunc("/api/GetStarredFiles", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		files, err := hs.app.GetFiles("", true, "")
		if err != nil {
			_ = json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		_ = json.NewEncoder(w).Encode(files)
	}))

	// Storage Allocation API
	mux.HandleFunc("/api/GetStorageAllocation", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		accounts, err := hs.app.GetAccounts()
		if err != nil {
			_ = json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		_ = json.NewEncoder(w).Encode(accounts)
	}))

	// Sync Tasks API
	mux.HandleFunc("/api/GetSyncTasks", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		tasks, err := hs.app.GetSyncTasks()
		if err != nil {
			_ = json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		_ = json.NewEncoder(w).Encode(tasks)
	}))

	// Version API
	mux.HandleFunc("/api/GetAppVersion", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode("1.0.0")
	}))

	// Sync / Stats API
	mux.HandleFunc("/api/sync", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			err := hs.app.syncMgr.SyncAllDrives()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Sync triggered successfully"})
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	// Virtual Drive API
	mux.HandleFunc("/api/GetVirtualDriveStatus", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status, err := hs.app.GetVirtualDriveStatus()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(status)
	}))
	mux.HandleFunc("/api/virtual-drive", authWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status, err := hs.app.GetVirtualDriveStatus()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(status)
	}))

	// Serve Frontend Static Assets with SPA fallback
	subFS, err := fs.Sub(assets, "frontend/dist")
	if err == nil {
		fileServer := http.FileServer(http.FS(subFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")

			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			// Check if file exists in static assets, if not fallback to index.html for SPA
			upath := r.URL.Path
			if !strings.HasPrefix(upath, "/") {
				upath = "/" + upath
			}
			upath = path.Clean(upath)
			f, openErr := subFS.Open(strings.TrimPrefix(upath, "/"))
			if openErr != nil {
				r.URL.Path = "/"
			} else {
				_ = f.Close()
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	addr := fmt.Sprintf("%s:%d", hs.host, hs.port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("==================================================")
	log.Printf("  Awd DriveRouter Headless Web Server active!")
	log.Printf("  Listening on: http://%s", addr)
	if hs.host == "0.0.0.0" || hs.host == "" {
		if addrs, err := net.InterfaceAddrs(); err == nil {
			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						log.Printf("  Local LAN Access: http://%s:%d", ipnet.IP.String(), hs.port)
					}
				}
			}
		}
	}
	if hs.apiKey != "" {
		log.Printf("  Authentication: API Key Protected")
	} else {
		log.Printf("  Authentication: Open Access (No API Key set)")
	}
	log.Printf("==================================================")

	// Channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Headless server error: %v", err)
		}
	}()

	<-stop
	log.Printf("Shutting down headless server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}
