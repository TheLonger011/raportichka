package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/TheLonger011/raportichka/internal/schedule"
)

func (h *Handler) handleScheduleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	go h.schedule.SyncNow()
	jsonResp(w, map[string]string{"status": "syncing"})
}

func (h *Handler) handleScheduleFiles(w http.ResponseWriter, r *http.Request) {
	sfiles, err := schedule.ListFiles(h.schedule.ScheduleDir())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	subfiles, err := schedule.ListFiles(h.schedule.SubstitutionsDir())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResp(w, map[string]interface{}{
		"schedule":      sfiles,
		"substitutions": subfiles,
	})
}

func (h *Handler) handleScheduleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Category string `json:"category"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var dir string
	switch req.Category {
	case "schedule":
		dir = h.schedule.ScheduleDir()
	case "substitutions":
		dir = h.schedule.SubstitutionsDir()
	default:
		http.Error(w, "unknown category", http.StatusBadRequest)
		return
	}

	safeName := filepath.Base(req.Name)
	if err := os.Remove(filepath.Join(dir, safeName)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"})
}
