package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func New(dsn, migrationsPath string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := RunMigrations(db, migrationsPath); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error { return s.db.Close() }

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS groups (
			id   SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE
		)`,
		`CREATE TABLE IF NOT EXISTS subjects (
			id       SERIAL PRIMARY KEY,
			name     TEXT NOT NULL,
			group_id INTEGER REFERENCES groups(id) ON DELETE CASCADE,
			UNIQUE (name, group_id)
		)`,
		`CREATE TABLE IF NOT EXISTS students (
			id        SERIAL PRIMARY KEY,
			full_name TEXT NOT NULL,
			group_id  INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
			UNIQUE (full_name, group_id)
		)`,
		`CREATE TABLE IF NOT EXISTS ocenki (
			id         SERIAL PRIMARY KEY,
			student_id INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE,
			subject_id INTEGER NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
			lesson_date DATE NOT NULL,
			grade      INT CHECK (grade >= 2 AND grade <= 5),
			status     TEXT DEFAULT 'present',
			UNIQUE (student_id, subject_id, lesson_date)
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id        SERIAL PRIMARY KEY,
			full_name TEXT NOT NULL,
			role      TEXT NOT NULL CHECK (role IN ('teacher', 'student')),
			group_id  INTEGER REFERENCES groups(id) ON DELETE SET NULL,
			password  TEXT NOT NULL,
			UNIQUE (full_name, role, group_id)
		)`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("createTables: %w", err)
		}
	}
	return nil
}
