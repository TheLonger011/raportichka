package main

import (
	"log"
	"strconv"

	"github.com/TheLonger011/raportichka/internal/config"
	"github.com/TheLonger011/raportichka/internal/storage/postgres"
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

	log.Println("Seeding database...")

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
		log.Printf("Error seeding groups/subjects/students: %v", err)
	} else {
		log.Println("Groups, subjects and students seeded successfully")
	}

	if err := storage.SeedUsers(); err != nil {
		log.Printf("Error seeding users: %v", err)
	} else {
		log.Println("Users seeded successfully")
	}

	log.Println("Seed completed!")
}
