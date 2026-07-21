package router

import (
	"driverouter/backend/db"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// SelectTargetAccounts decides which active account(s) should receive the upload based on the active strategy.
func SelectTargetAccounts(database *db.DB, manualAccountID string) ([]db.AccountRecord, string, error) {
	accounts, err := database.GetAccounts()
	if err != nil {
		return nil, "", fmt.Errorf("failed to retrieve accounts: %w", err)
	}

	var activeAccounts []db.AccountRecord
	for _, a := range accounts {
		if a.Active {
			activeAccounts = append(activeAccounts, a)
		}
	}

	if len(activeAccounts) == 0 {
		return nil, "", errors.New("no active cloud accounts connected. please connect an account in settings first")
	}

	strategy, err := database.GetSetting("upload_strategy")
	if err != nil || strategy == "" {
		strategy = "round_robin"
	}

	// Override strategy if manual account is specified
	if manualAccountID != "" {
		strategy = "manual"
	}

	switch strategy {
	case "manual":
		for _, a := range activeAccounts {
			if a.ID == manualAccountID {
				return []db.AccountRecord{a}, "manual", nil
			}
		}
		return nil, "", fmt.Errorf("selected manual account %s is not active or connected", manualAccountID)

	case "max_free":
		var selected db.AccountRecord
		var maxFree int64 = -1
		for _, a := range activeAccounts {
			free := a.TotalSpace - a.UsedSpace
			if free > maxFree {
				maxFree = free
				selected = a
			}
		}
		return []db.AccountRecord{selected}, "max_free", nil

	case "weighted_round_robin":
		var candidates []db.AccountRecord
		for _, a := range activeAccounts {
			// calculate weight in GB
			weight := int(a.TotalSpace / (1024 * 1024 * 1024))
			if weight < 1 {
				weight = 1
			}
			if weight > 1000 {
				weight = 1000
			}
			for i := 0; i < weight; i++ {
				candidates = append(candidates, a)
			}
		}
		counterStr, _ := database.GetSetting("weighted_counter")
		counter := 0
		fmt.Sscanf(counterStr, "%d", &counter)
		selected := candidates[counter%len(candidates)]
		_ = database.SaveSetting("weighted_counter", fmt.Sprintf("%d", counter+1))
		return []db.AccountRecord{selected}, "weighted_round_robin", nil

	case "least_used":
		var selected db.AccountRecord
		var minRatio float64 = 2.0
		for _, a := range activeAccounts {
			ratio := 1.0
			if a.TotalSpace > 0 {
				ratio = float64(a.UsedSpace) / float64(a.TotalSpace)
			}
			if ratio < minRatio {
				minRatio = ratio
				selected = a
			}
		}
		return []db.AccountRecord{selected}, "least_used", nil

	case "custom_order":
		orderStr, _ := database.GetSetting("custom_account_order")
		if orderStr != "" {
			parts := strings.Split(orderStr, ",")
			for _, id := range parts {
				trimmed := strings.TrimSpace(id)
				for _, a := range activeAccounts {
					if a.ID == trimmed {
						return []db.AccountRecord{a}, "custom_order", nil
					}
				}
			}
		}
		return []db.AccountRecord{activeAccounts[0]}, "custom_order", nil

	case "mirror":
		// Upload to all active accounts
		return activeAccounts, "mirror", nil

	case "round_robin":
		fallthrough
	default:
		lastAccountID, err := database.GetSetting("last_upload_account_id")
		if err != nil {
			lastAccountID = ""
		}

		// Find the index of the last used account
		lastIndex := -1
		for i, a := range activeAccounts {
			if a.ID == lastAccountID {
				lastIndex = i
				break
			}
		}

		// Choose the next one
		nextIndex := (lastIndex + 1) % len(activeAccounts)
		selected := activeAccounts[nextIndex]

		// Save the chosen account ID for next time
		_ = database.SaveSetting("last_upload_account_id", selected.ID)

		return []db.AccountRecord{selected}, "round_robin", nil
	}
}

// PhysicalIDsMap wraps a helper JSON struct stored in the files table `physical_id` column.
// Format: {"account_id": "physical_file_id"}
type PhysicalIDsMap map[string]string

func SerializePhysicalIDs(m PhysicalIDsMap) (string, error) {
	b, err := json.Marshal(m)
	return string(b), err
}

func DeserializePhysicalIDs(s string) (PhysicalIDsMap, error) {
	var m PhysicalIDsMap
	if s == "" {
		return make(PhysicalIDsMap), nil
	}
	// Try parsing as JSON map
	err := json.Unmarshal([]byte(s), &m)
	if err != nil {
		// Fallback for older format if raw string
		m = make(PhysicalIDsMap)
		return m, err
	}
	return m, nil
}
