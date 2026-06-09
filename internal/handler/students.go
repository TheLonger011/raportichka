package handler

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) handleAddStudent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		FullName string `json:"full_name"`
		GroupID  int    `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.FullName == "" {
		http.Error(w, "full_name is required", http.StatusBadRequest)
		return
	}
	if req.GroupID == 0 {
		http.Error(w, "group_id is required", http.StatusBadRequest)
		return
	}
	id, err := h.storage.AddStudent(req.FullName, req.GroupID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.storage.SeedUsers(); err != nil {
		_ = err
	}
	jsonResp(w, map[string]interface{}{"id": id, "status": "ok"})
}

func (h *Handler) handleDeleteStudent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		StudentID int `json:"student_id"`
		GroupID   int `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.StudentID == 0 {
		http.Error(w, "student_id is required", http.StatusBadRequest)
		return
	}
	if err := h.storage.DeleteStudent(req.StudentID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"})
}
