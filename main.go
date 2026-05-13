package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"raportichka/internal/storage/postgres"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	date, err := os.ReadFile("config/local.yaml")
	if err != nil {
		log.Fatal(err)
	}

	var config struct {
		StoragePath string `yaml:"storage_path"`
	}

	err = yaml.Unmarshal(date, &config)
	if err != nil {
		log.Fatal(err)
	}

	storagePath := config.StoragePath

	storage, err := postgres.New(storagePath)
	if err != nil {
		log.Fatal(err)
	}
	defer storage.Close()

	if err := loadStudentsFromFile(storage, "students.txt"); err != nil {
		log.Printf("Warning: failed to load students: %v", err)
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("."))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/api/grades", func(w http.ResponseWriter, r *http.Request) {
		yearStr := r.URL.Query().Get("year")
		monthStr := r.URL.Query().Get("month")

		year, err := strconv.Atoi(yearStr)
		if err != nil || year == 0 {
			year = 2026
		}

		month, err := strconv.Atoi(monthStr)
		if err != nil || month == 0 {
			month = 5
		}

		grades, err := storage.GetGrade(year, month)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(grades)
	})

	http.HandleFunc("/api/set-grade", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			StudentID int    `json:"student_id"`
			Date      string `json:"date"`
			Grade     *int   `json:"grade"`
			Status    string `json:"status"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Grade != nil && *req.Grade >= 2 && *req.Grade <= 5 {
			err = storage.SetGrade(req.StudentID, req.Date, *req.Grade)
		} else if req.Status == "absent" || req.Status == "excused" {
			err = storage.SetAbsent(req.StudentID, req.Date, req.Status)
		} else {
			err = storage.SetAbsent(req.StudentID, req.Date, "")
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	log.Println("Сервер запущен на http://localhost:8800")
	log.Fatal(http.ListenAndServe(":8800", nil))
}

func loadStudentsFromFile(storage *postgres.Storage, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var students []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			students = append(students, name)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return storage.LoadStudentsFromFile(students)
}
