package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func (h *Handler) handleVedomost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		GroupID     int    `json:"group_id"`
		Year        int    `json:"year"`
		Month       int    `json:"month"`
		SubjectIDs  []int  `json:"subject_ids"`
		TotalHours  int    `json:"total_hours"`
		HeadTeacher string `json:"head_teacher"`
		DeptHead    string `json:"dept_head"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Year == 0 {
		req.Year = 2026
	}
	if req.Month == 0 {
		req.Month = 5
	}
	if req.TotalHours == 0 {
		req.TotalHours = 120
	}

	result, err := h.vedomost.Generate(r.Context(), req.GroupID, req.Year, req.Month,
		req.SubjectIDs, req.TotalHours, req.HeadTeacher, req.DeptHead)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+result.Filename+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(result.Data)))
	w.Write(result.Data)
}
