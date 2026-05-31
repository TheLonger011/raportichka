package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/TheLonger011/raportichka/internal/auth"
	"github.com/TheLonger011/raportichka/internal/config"
	"github.com/TheLonger011/raportichka/internal/handler"
	"github.com/TheLonger011/raportichka/internal/schedule"
	"github.com/TheLonger011/raportichka/internal/storage/postgres"
	"github.com/TheLonger011/raportichka/internal/vedomost"
)

const (
	scheduleDir      = "downloads/schedule"
	substitutionsDir = "downloads/substitutions"
)

func main() {
	cfg, err := config.Load("config/local.yaml")
	if err != nil {
		log.Fatal(err)
	}

	storage, err := postgres.New(cfg.StoragePath, "migrations")
	if err != nil {
		log.Fatal(err)
	}
	defer storage.Close()

	if cfg.SeedOnStart {
		if isEmpty, _ := storage.IsEmpty(); isEmpty {
			log.Println("Seeding database with test data...")
			if err := seedData(storage); err != nil {
				log.Printf("Seed warning: %v", err)
			} else {
				log.Println("Database seeded successfully")
			}
		}
	}

	sessions := auth.NewSessionStore()

	dl := schedule.New(scheduleDir, substitutionsDir, cfg.ScheduleKey, cfg.SubstitutionsKey, cfg.SyncIntervalHours)
	dl.Start()

	vs := vedomost.NewService(storage)

	h := handler.New(storage, sessions, dl, vs)

	mux := http.NewServeMux()
	h.Register(mux, scheduleDir, substitutionsDir)

	srv := &http.Server{
		Addr:         ":8800",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("Server starting on http://localhost:8800")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func seedData(storage *postgres.Storage) error {
	groupSubjects := map[string][]string{
		"ИС-2-01": {"Арх. апп. средств", "Физ-ра", "МДК 05.01", "МДК 08.01", "Основы алгоритм.", "Теория вероят. и мат. статистика", "Численные методы", "Операционные среды"},
		"К-2-80":  {"МДК 02.01", "МДК 01.01", "Материал", "Ин. язык", "Инж. граф.", "БЖД"},
		"ГД-3-03": {"МДК 02.02", "Физ-ра", "Ин. язык", "МДК 01.02", "Осн. материал."},
		"ИС-2-02": {"Арх. апп. средств", "Физ-ра", "МДК 05.01", "МДК 08.01", "Основы алгоритм.", "Теория вероят. и мат. статистика"},
		"ЭМ-2-02": {"Электро техника", "Физ-ра", "Ин. язык"},
		"ПС-1-17": {"Адм.география", "Информатика", "Физика", "ОПД", "История Коми", "Математика", "Русский язык", "Химия", "История", "Родная литература", "География", "Обществознание"},
	}

	studentsPerGroup := make(map[string][]string, len(groupSubjects))
	for gname := range groupSubjects {
		students := make([]string, 9)
		for i := range students {
			students[i] = "Ученик " + strconv.Itoa(i+1)
		}
		studentsPerGroup[gname] = students
	}

	if err := storage.SeedData(groupSubjects, studentsPerGroup); err != nil {
		return err
	}

	return storage.SeedUsers()
}
