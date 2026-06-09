package handler

import (
	"encoding/json"
	"net/http"

	"github.com/TheLonger011/raportichka/internal/auth"
	"github.com/TheLonger011/raportichka/internal/schedule"
	"github.com/TheLonger011/raportichka/internal/storage/postgres"
	"github.com/TheLonger011/raportichka/internal/vedomost"
)

type Handler struct {
	storage  *postgres.Storage
	sessions *auth.SessionStore
	schedule *schedule.Downloader
	vedomost *vedomost.Service
}

func New(
	storage *postgres.Storage,
	sessions *auth.SessionStore,
	dl *schedule.Downloader,
	vs *vedomost.Service,
) *Handler {
	return &Handler{
		storage:  storage,
		sessions: sessions,
		schedule: dl,
		vedomost: vs,
	}
}

func (h *Handler) Register(mux *http.ServeMux, scheduleDir, substitutionsDir string) {
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/files/schedule/", http.StripPrefix("/files/schedule/", http.FileServer(http.Dir(scheduleDir))))
	mux.Handle("/files/substitutions/", http.StripPrefix("/files/substitutions/", http.FileServer(http.Dir(substitutionsDir))))
	mux.Handle("/pages/", http.StripPrefix("/pages/", http.FileServer(http.Dir("pages"))))

	mux.HandleFunc("/login", h.handleLoginPage)
	mux.HandleFunc("/logout", h.handleLogout)
	mux.HandleFunc("/", auth.RequireTeacher(h.sessions, h.handlePage("pages/groups/index.html")))
	mux.HandleFunc("/grades", auth.RequireTeacher(h.sessions, h.handlePage("pages/grades/index.html")))
	mux.HandleFunc("/schedule", auth.RequireAuth(h.sessions, h.handlePage("pages/schedule/index.html")))
	mux.HandleFunc("/student", auth.RequireStudent(h.sessions, h.handlePage("pages/student/index.html")))
	mux.HandleFunc("/vedomost", auth.RequireTeacher(h.sessions, h.handlePage("pages/vedomost/index.html")))

	mux.HandleFunc("/api/login", h.handleAPILogin)
	mux.HandleFunc("/api/me", h.handleAPIMe)
	mux.HandleFunc("/api/groups-public", h.handleGroupsPublic)

	mux.HandleFunc("/api/groups", auth.RequireTeacher(h.sessions, h.handleGroups))
	mux.HandleFunc("/api/subjects", auth.RequireAuth(h.sessions, h.handleSubjects))
	mux.HandleFunc("/api/students", auth.RequireTeacher(h.sessions, h.handleStudents))
	mux.HandleFunc("/api/grades", auth.RequireTeacher(h.sessions, h.handleGrades))
	mux.HandleFunc("/api/set-grade", auth.RequireTeacher(h.sessions, h.handleSetGrade))

	mux.HandleFunc("/api/student/grades", auth.RequireStudent(h.sessions, h.handleStudentGrades))

	mux.HandleFunc("/api/schedule/sync", h.handleScheduleSync)
	mux.HandleFunc("/api/schedule/files", h.handleScheduleFiles)
	mux.HandleFunc("/api/schedule/delete", h.handleScheduleDelete)

	mux.HandleFunc("/api/vedomost/generate", auth.RequireTeacher(h.sessions, h.handleVedomost))

	mux.HandleFunc("/api/students/add", auth.RequireTeacher(h.sessions, h.handleAddStudent))
	mux.HandleFunc("/api/students/delete", auth.RequireTeacher(h.sessions, h.handleDeleteStudent))
}

func (h *Handler) handlePage(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path)
	}
}

func jsonResp(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
