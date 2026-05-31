package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func (h *Handler) handleGroupsPublic(w http.ResponseWriter, r *http.Request) {
	groups, err := h.storage.GetGroups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResp(w, groups)
}

func (h *Handler) handleGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.storage.GetGroups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResp(w, groups)
}

func (h *Handler) handleSubjects(w http.ResponseWriter, r *http.Request) {
	gid, _ := strconv.Atoi(r.URL.Query().Get("group_id"))
	subjects, err := h.storage.GetSubjectsByGroup(gid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResp(w, subjects)
}

func (h *Handler) handleStudents(w http.ResponseWriter, r *http.Request) {
	gid, _ := strconv.Atoi(r.URL.Query().Get("group_id"))
	students, err := h.storage.GetStudentsByGroup(gid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResp(w, students)
}

func (h *Handler) handleGrades(w http.ResponseWriter, r *http.Request) {
	gid, _ := strconv.Atoi(r.URL.Query().Get("group_id"))
	sid, _ := strconv.Atoi(r.URL.Query().Get("subject_id"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	month, _ := strconv.Atoi(r.URL.Query().Get("month"))
	if year == 0 {
		year = 2026
	}
	if month == 0 {
		month = 5
	}

	grades, err := h.storage.GetGrades(gid, sid, year, month)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResp(w, grades)
}

func (h *Handler) handleSetGrade(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		StudentID int    `json:"student_id"`
		SubjectID int    `json:"subject_id"`
		Date      string `json:"date"`
		Grade     *int   `json:"grade"`
		Status    string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var err error
	switch {
	case req.Grade != nil && *req.Grade >= 2 && *req.Grade <= 5:
		err = h.storage.SetGrade(req.StudentID, req.SubjectID, req.Date, *req.Grade)
	case req.Status == "absent" || req.Status == "excused":
		err = h.storage.SetAbsent(req.StudentID, req.SubjectID, req.Date, req.Status)
	default:
		err = h.storage.ClearGrade(req.StudentID, req.SubjectID, req.Date)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"})
}

func (h *Handler) handleStudentGrades(w http.ResponseWriter, r *http.Request) {
	sess := h.sessions.GetSession(r)
	if sess.GroupID == nil {
		http.Error(w, "no group", http.StatusBadRequest)
		return
	}
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	month, _ := strconv.Atoi(r.URL.Query().Get("month"))
	if year == 0 {
		year = 2026
	}
	if month == 0 {
		month = 5
	}

	studentID, err := h.storage.GetStudentIDByUserID(sess.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := h.storage.GetStudentGradesAllSubjects(studentID, *sess.GroupID, year, month)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResp(w, data)
}
