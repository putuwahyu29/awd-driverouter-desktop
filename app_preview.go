package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"driverouter/backend/db"
	"driverouter/backend/router"
	"driverouter/backend/sync"
)

// PreviewFile downloads a file to memory and returns content (as base64 or text) for preview.
func (a *App) PreviewFile(virtualID string) (map[string]interface{}, error) {
	f, err := a.database.GetFile(virtualID)
	if err != nil {
		return nil, err
	}

	if f.IsFolder {
		return nil, fmt.Errorf("directories cannot be previewed")
	}

	physicalMap, err := router.DeserializePhysicalIDs(f.PhysicalID)
	if err != nil || len(physicalMap) == 0 {
		return nil, fmt.Errorf("no physical file links found")
	}

	var activeAcc db.AccountRecord
	var physID string
	foundAccount := false

	accounts, _ := a.database.GetAccounts()
	for accID, pID := range physicalMap {
		for _, acc := range accounts {
			if acc.ID == accID && acc.Active {
				physID = pID
				activeAcc = acc
				foundAccount = true
				break
			}
		}
		if foundAccount {
			break
		}
	}

	if !foundAccount {
		return nil, fmt.Errorf("no active account holds a copy of this file")
	}

	p, err := sync.FetchActiveProviderClient(a.database, activeAcc, nil)
	if err != nil {
		return nil, err
	}

	reader, err := p.DownloadFile(physID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	ext := ""
	if idx := strings.LastIndex(f.Name, "."); idx != -1 {
		ext = strings.ToLower(f.Name[idx:])
	}

	isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" || ext == ".bmp" || ext == ".ico"
	isPdf := ext == ".pdf"
	isAudio := ext == ".mp3" || ext == ".wav" || ext == ".ogg" || ext == ".m4a" || ext == ".flac" || ext == ".aac"
	isVideo := ext == ".mp4" || ext == ".webm" || ext == ".ogg" || ext == ".ogv" || ext == ".mov" || ext == ".mkv"

	// Text extensions
	isText := ext == ".txt" || ext == ".log" || ext == ".json" || ext == ".md" || ext == ".csv" ||
		ext == ".js" || ext == ".ts" || ext == ".html" || ext == ".css" || ext == ".go" || ext == ".py" ||
		ext == ".xml" || ext == ".yaml" || ext == ".yml" || ext == ".toml" || ext == ".ini" || ext == ".sh" ||
		ext == ".bat" || ext == ".sql"

	res := map[string]interface{}{
		"success":  true,
		"fileName": f.Name,
		"ext":      ext,
	}

	if isImage {
		b64 := base64.StdEncoding.EncodeToString(data)
		mime := "image/jpeg"
		if ext == ".png" {
			mime = "image/png"
		} else if ext == ".gif" {
			mime = "image/gif"
		} else if ext == ".webp" {
			mime = "image/webp"
		} else if ext == ".svg" {
			mime = "image/svg+xml"
		} else if ext == ".bmp" {
			mime = "image/bmp"
		} else if ext == ".ico" {
			mime = "image/x-icon"
		}
		res["base64"] = "data:" + mime + ";base64," + b64
	} else if isPdf {
		b64 := base64.StdEncoding.EncodeToString(data)
		res["base64"] = "data:application/pdf;base64," + b64
	} else if isAudio {
		b64 := base64.StdEncoding.EncodeToString(data)
		mime := "audio/mpeg"
		if ext == ".wav" {
			mime = "audio/wav"
		} else if ext == ".ogg" {
			mime = "audio/ogg"
		} else if ext == ".m4a" {
			mime = "audio/mp4"
		} else if ext == ".flac" {
			mime = "audio/flac"
		} else if ext == ".aac" {
			mime = "audio/aac"
		}
		res["base64"] = "data:" + mime + ";base64," + b64
	} else if isVideo {
		b64 := base64.StdEncoding.EncodeToString(data)
		mime := "video/mp4"
		if ext == ".webm" {
			mime = "video/webm"
		} else if ext == ".ogg" || ext == ".ogv" {
			mime = "video/ogg"
		} else if ext == ".mov" {
			mime = "video/quicktime"
		} else if ext == ".mkv" {
			mime = "video/x-matroska"
		}
		res["base64"] = "data:" + mime + ";base64," + b64
	} else if isText {
		res["text"] = string(data)
	} else {
		return nil, fmt.Errorf("UNSUPPORTED_TYPE")
	}

	return res, nil
}
