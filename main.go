package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"raportichka/internal/schedule"
	"raportichka/internal/storage/postgres"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	StoragePath       string `yaml:"storage_path"`
	ScheduleKey       string `yaml:"schedule_key"`
	SubstitutionsKey  string `yaml:"substitutions_key"`
	SyncIntervalHours int    `yaml:"sync_interval_hours"`
}

const (
	scheduleDir      = "downloads/schedule"
	substitutionsDir = "downloads/substitutions"
)

func main() {
	data, err := os.ReadFile("config/local.yaml")
	if err != nil {
		log.Fatal(err)
	}
	var cfg Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatal(err)
	}
	if cfg.SyncIntervalHours == 0 {
		cfg.SyncIntervalHours = 6
	}

	storage, err := postgres.New(cfg.StoragePath)
	if err != nil {
		log.Fatal(err)
	}
	defer storage.Close()

	groupSubjects := map[string][]string{
		"ИС-2-01": {"Арх. апп. средств", "Физ-ра", "МДК 05.01", "МДК 08.01", "Основы алгоритм.", "Теория вероят. и мат. статистика", "численные методы", "Операционные среды"},
		"К-2-80":  {"МДК 02.01", "МДК 01.01", "Материал", "Ин. язык", "Инж. граф.", "БЖД"},
		"ГД-3-03": {"МДК 02.02", "Физ-ра", "Ин. язык", "МДК 01.02", "Осн. материал."},
		"ИС-2-02": {"Арх. апп. средств", "Физ-ра", "МДК 05.01", "МДК 08.01", "Основы алгоритм.", "Теория вероят. и мат. статистика"},
	}
	studentsPerGroup := map[string][]string{}
	for gname := range groupSubjects {
		students := make([]string, 5)
		for i := range students {
			students[i] = "Ученик " + strconv.Itoa(i+1)
		}
		studentsPerGroup[gname] = students
	}
	if err := storage.SeedData(groupSubjects, studentsPerGroup); err != nil {
		log.Printf("Seed warning: %v", err)
	}
	if err := storage.SeedUsers(); err != nil {
		log.Printf("SeedUsers warning: %v", err)
	}

	sessions := NewSessionStore()

	dl := schedule.New(scheduleDir, substitutionsDir, cfg.ScheduleKey, cfg.SubstitutionsKey, cfg.SyncIntervalHours)
	dl.Start()

	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/files/schedule/", http.StripPrefix("/files/schedule/", http.FileServer(http.Dir(scheduleDir))))
	mux.Handle("/files/substitutions/", http.StripPrefix("/files/substitutions/", http.FileServer(http.Dir(substitutionsDir))))
	mux.Handle("/pages/", http.StripPrefix("/pages/", http.FileServer(http.Dir("pages"))))

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		sess := sessions.GetSession(r)
		if sess != nil {
			if sess.Role == "student" {
				http.Redirect(w, r, "/student", http.StatusFound)
			} else {
				http.Redirect(w, r, "/", http.StatusFound)
			}
			return
		}
		http.ServeFile(w, r, "pages/login/index.html")
	})

	mux.HandleFunc("/", RequireTeacher(sessions, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pages/groups/index.html")
	}))

	mux.HandleFunc("/grades", RequireTeacher(sessions, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pages/grades/index.html")
	}))

	mux.HandleFunc("/schedule", RequireAuth(sessions, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pages/schedule/index.html")
	}))

	mux.HandleFunc("/student", RequireStudent(sessions, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pages/student/index.html")
	}))

	mux.HandleFunc("/vedomost", RequireTeacher(sessions, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "pages/vedomost/index.html")
	}))

	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err == nil {
			sessions.Delete(c.Value)
		}
		sessions.ClearCookie(w)
		http.Redirect(w, r, "/login", http.StatusFound)
	})

	mux.HandleFunc("/api/groups-public", func(w http.ResponseWriter, r *http.Request) {
		groups, err := storage.GetGroups()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		jsonResp(w, groups)
	})

	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		var req struct {
			Role     string `json:"role"`
			FullName string `json:"full_name"`
			Password string `json:"password"`
			GroupID  *int   `json:"group_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", 400)
			return
		}

		role := postgres.Role(req.Role)
		user, err := storage.Login(req.FullName, req.Password, role, req.GroupID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		sess := &SessionData{
			UserID:    user.ID,
			FullName:  user.FullName,
			Role:      string(user.Role),
			GroupID:   user.GroupID,
			GroupName: user.GroupName,
		}
		sid := sessions.Create(sess)
		sessions.SetCookie(w, sid)

		redirect := "/"
		if user.Role == postgres.RoleStudent {
			redirect = "/student"
		}
		jsonResp(w, map[string]string{"redirect": redirect})
	})

	mux.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		sess := sessions.GetSession(r)
		if sess == nil {
			http.Error(w, "unauthorized", 401)
			return
		}
		jsonSession(w, sess)
	})

	mux.HandleFunc("/api/groups", RequireTeacher(sessions, func(w http.ResponseWriter, r *http.Request) {
		groups, err := storage.GetGroups()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		jsonResp(w, groups)
	}))

	mux.HandleFunc("/api/subjects", RequireAuth(sessions, func(w http.ResponseWriter, r *http.Request) {
		gid, _ := strconv.Atoi(r.URL.Query().Get("group_id"))
		subjects, err := storage.GetSubjectsByGroup(gid)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		jsonResp(w, subjects)
	}))

	mux.HandleFunc("/api/students", RequireTeacher(sessions, func(w http.ResponseWriter, r *http.Request) {
		gid, _ := strconv.Atoi(r.URL.Query().Get("group_id"))
		students, err := storage.GetStudentsByGroup(gid)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		jsonResp(w, students)
	}))

	mux.HandleFunc("/api/grades", RequireTeacher(sessions, func(w http.ResponseWriter, r *http.Request) {
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
		grades, err := storage.GetGrades(gid, sid, year, month)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		jsonResp(w, grades)
	}))

	mux.HandleFunc("/api/set-grade", RequireTeacher(sessions, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
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
			http.Error(w, err.Error(), 400)
			return
		}
		if req.Grade != nil && *req.Grade >= 2 && *req.Grade <= 5 {
			err = storage.SetGrade(req.StudentID, req.SubjectID, req.Date, *req.Grade)
		} else if req.Status == "absent" || req.Status == "excused" {
			err = storage.SetAbsent(req.StudentID, req.SubjectID, req.Date, req.Status)
		} else {
			err = storage.ClearGrade(req.StudentID, req.SubjectID, req.Date)
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "ok"})
	}))

	mux.HandleFunc("/api/student/grades", RequireStudent(sessions, func(w http.ResponseWriter, r *http.Request) {
		sess := sessions.GetSession(r)
		if sess.GroupID == nil {
			http.Error(w, "no group", 400)
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

		studentID, err := storage.GetStudentIDByUserID(sess.UserID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		data, err := storage.GetStudentGradesAllSubjects(studentID, *sess.GroupID, year, month)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		jsonResp(w, data)
	}))

	mux.HandleFunc("/api/schedule/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		go dl.SyncNow()
		jsonResp(w, map[string]string{"status": "syncing"})
	})

	mux.HandleFunc("/api/schedule/files", func(w http.ResponseWriter, r *http.Request) {
		sfiles, err := schedule.ListFiles(scheduleDir)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		subfiles, err := schedule.ListFiles(substitutionsDir)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]interface{}{
			"schedule":      sfiles,
			"substitutions": subfiles,
		})
	})

	mux.HandleFunc("/api/schedule/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		var req struct {
			Category string `json:"category"`
			Name     string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var dir string
		if req.Category == "schedule" {
			dir = scheduleDir
		} else if req.Category == "substitutions" {
			dir = substitutionsDir
		} else {
			http.Error(w, "unknown category", 400)
			return
		}
		safeName := filepath.Base(req.Name)
		if err := os.Remove(filepath.Join(dir, safeName)); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/vedomost/generate", RequireTeacher(sessions, makeHandleVedomost(storage)))

	log.Println("http://localhost:8800")
	log.Fatal(http.ListenAndServe(":8800", mux))
}

func jsonResp(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(v)
}
