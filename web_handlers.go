package main

import (
	"archive/zip"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"driverouter/backend/db"
	"driverouter/backend/router"
	"driverouter/backend/sync"
)

func (ws *WebServer) generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:12]
}

func (ws *WebServer) handleLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(logoPngBytes)
}

func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	ws.serve404Page(w)
}

func (ws *WebServer) getFileStream(virtualID string) (io.ReadCloser, db.FileRecord, error) {
	f, err := ws.App.database.GetFile(virtualID)
	if err != nil {
		return nil, db.FileRecord{}, fmt.Errorf("file not found: %w", err)
	}

	if f.IsFolder {
		return nil, f, fmt.Errorf("cannot stream folder directly")
	}

	physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
	if err != nil || len(physicalMap) == 0 {
		return nil, f, fmt.Errorf("no physical copies linked to file")
	}

	accounts, _ := ws.App.database.GetAccounts()
	var activeAcc db.AccountRecord
	var physID string
	found := false

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
		return nil, f, fmt.Errorf("no active account found for file")
	}

	p, err := sync.FetchActiveProviderClient(ws.App.database, activeAcc, nil)
	if err != nil {
		return nil, f, fmt.Errorf("failed to initialize provider client: %w", err)
	}

	reader, err := p.DownloadFile(physID)
	if err != nil {
		return nil, f, fmt.Errorf("failed to download file: %w", err)
	}

	return reader, f, nil
}

func (ws *WebServer) handleShare(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/share/")
	id = strings.Split(id, "/")[0] // Extract base share ID

	var item *WebShareItem
	ws.Mu.Lock()
	for i := range ws.SharedItems {
		if ws.SharedItems[i].ID == id {
			item = &ws.SharedItems[i]
			item.AccessCount++
			break
		}
	}
	ws.Mu.Unlock()

	if item == nil {
		ws.serve404Page(w)
		return
	}

	ws.SaveShares()

	// Check password
	if item.Password != "" {
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			passInput := r.FormValue("p")
			if passInput == item.Password {
				// Set authentication cookie
				http.SetCookie(w, &http.Cookie{
					Name:     "share_auth_" + id,
					Value:    item.Password,
					Path:     "/",
					MaxAge:   86400, // 24 hours
					HttpOnly: true,
				})
				http.Redirect(w, r, "/share/"+id, http.StatusSeeOther)
				return
			} else {
				ws.servePasswordPage(w, item.Name, id, true)
				return
			}
		}

		// Check cookie
		cookie, err := r.Cookie("share_auth_" + id)
		if err != nil || cookie.Value != item.Password {
			ws.servePasswordPage(w, item.Name, id, false)
			return
		}
	}

	// Check sub-file preview request
	subFileID := r.URL.Query().Get("file")
	if subFileID != "" && item.Type == "folder" {
		subFile, err := ws.App.database.GetFile(subFileID)
		if err == nil {
			mockItem := &WebShareItem{
				ID:        id + "/" + subFile.ID,
				Name:      subFile.Name,
				Size:      subFile.Size,
				VirtualID: subFile.ID,
				Type:      "file",
			}
			ws.serveFilePage(w, mockItem)
			return
		}
	}

	if item.Type == "folder" {
		ws.serveFolderPage(w, item)
	} else {
		ws.serveFilePage(w, item)
	}
}

func (ws *WebServer) handleDownload(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/download/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}
	shareID := parts[0]

	var item *WebShareItem
	ws.Mu.Lock()
	for i := range ws.SharedItems {
		if ws.SharedItems[i].ID == shareID {
			item = &ws.SharedItems[i]
			break
		}
	}
	ws.Mu.Unlock()

	if item == nil {
		http.Error(w, "Link expired or not found", http.StatusNotFound)
		return
	}

	// Check password via cookie
	if item.Password != "" {
		cookie, err := r.Cookie("share_auth_" + shareID)
		if err != nil || cookie.Value != item.Password {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	targetVirtualID := item.VirtualID
	if item.Type == "folder" && len(parts) >= 2 {
		targetVirtualID = parts[1]
	}

	reader, fRec, err := ws.getFileStream(targetVirtualID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch stream: %v", err), http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	contentType := "application/octet-stream"
	if ext := filepath.Ext(fRec.Name); ext != "" {
		if t := mime.TypeByExtension(ext); t != "" {
			contentType = t
		}
	}

	disposition := "attachment"
	if r.URL.Query().Get("inline") == "1" {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, fRec.Name))
	if fRec.Size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fRec.Size))
	}

	_, _ = io.Copy(w, reader)
}

func (ws *WebServer) handleBatchDownload(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/download_batch/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}
	shareID := parts[0]

	var item *WebShareItem
	ws.Mu.Lock()
	for i := range ws.SharedItems {
		if ws.SharedItems[i].ID == shareID {
			item = &ws.SharedItems[i]
			break
		}
	}
	ws.Mu.Unlock()

	if item == nil || item.Type != "folder" {
		http.Error(w, "Invalid batch share target", http.StatusBadRequest)
		return
	}

	if item.Password != "" {
		cookie, err := r.Cookie("share_auth_" + shareID)
		if err != nil || cookie.Value != item.Password {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", item.Name))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	ws.addFolderToZip(zipWriter, item.VirtualID, item.Name)
}

func (ws *WebServer) addFolderToZip(zw *zip.Writer, folderID string, relativePath string) {
	files, err := ws.App.database.GetFiles(folderID, false, "")
	if err != nil {
		return
	}

	for _, file := range files {
		itemPath := filepath.Join(relativePath, file.Name)
		if file.IsFolder {
			ws.addFolderToZip(zw, file.ID, itemPath)
		} else {
			reader, _, err := ws.getFileStream(file.ID)
			if err != nil {
				continue
			}
			fWriter, err := zw.Create(itemPath)
			if err == nil {
				_, _ = io.Copy(fWriter, reader)
			}
			_ = reader.Close()
		}
	}
}
