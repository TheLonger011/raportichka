package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := sql.Open("postgres", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &Storage{db: db}, nil
}

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
			id       SERIAL PRIMARY KEY,
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
			id         SERIAL PRIMARY KEY,
			full_name  TEXT NOT NULL,
			role       TEXT NOT NULL CHECK (role IN ('teacher', 'student')),
			group_id   INTEGER REFERENCES groups(id) ON DELETE SET NULL,
			password   TEXT NOT NULL,
			UNIQUE (full_name, role, group_id)
		)`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("failed to exec: %w", err)
		}
	}
	return nil
}

func (s *Storage) Close() error { return s.db.Close() }

func (s *Storage) GetGroups() ([]map[string]interface{}, error) {
	rows, err := s.db.Query(`SELECT id, name FROM groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		rows.Scan(&id, &name)
		result = append(result, map[string]interface{}{"id": id, "name": name})
	}
	return result, nil
}

func (s *Storage) AddGroup(name string) (int, error) {
	var id int
	err := s.db.QueryRow(
		`INSERT INTO groups (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name RETURNING id`,
		name,
	).Scan(&id)
	return id, err
}

func (s *Storage) GetSubjectsByGroup(groupID int) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(`SELECT id, name FROM subjects WHERE group_id=$1 ORDER BY name`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		rows.Scan(&id, &name)
		result = append(result, map[string]interface{}{"id": id, "name": name})
	}
	return result, nil
}

func (s *Storage) AddSubject(name string, groupID int) (int, error) {
	var id int
	err := s.db.QueryRow(
		`INSERT INTO subjects (name, group_id) VALUES ($1, $2) ON CONFLICT (name, group_id) DO UPDATE SET name=EXCLUDED.name RETURNING id`,
		name, groupID,
	).Scan(&id)
	return id, err
}

func (s *Storage) GetStudentsByGroup(groupID int) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(
		`SELECT id, full_name FROM students WHERE group_id=$1 ORDER BY id`, groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		rows.Scan(&id, &name)
		result = append(result, map[string]interface{}{"id": id, "full_name": name})
	}
	return result, nil
}

func (s *Storage) AddStudent(fullName string, groupID int) (int, error) {
	var id int
	err := s.db.QueryRow(
		`INSERT INTO students (full_name, group_id) VALUES ($1, $2)
		 ON CONFLICT (full_name, group_id) DO UPDATE SET full_name=EXCLUDED.full_name RETURNING id`,
		fullName, groupID,
	).Scan(&id)
	return id, err
}

func (s *Storage) GetGrades(groupID, subjectID, year, month int) ([]map[string]interface{}, error) {
	firstDay := fmt.Sprintf("%d-%02d-01", year, month)

	var daysCount int
	err := s.db.QueryRow(
		`SELECT EXTRACT(DAY FROM ($1::date + INTERVAL '1 month - 1 day'))::int`, firstDay,
	).Scan(&daysCount)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(
		`SELECT id, full_name FROM students WHERE group_id=$1 ORDER BY id`, groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type gradeInfo struct {
		grade  *int
		status string
	}

	var result []map[string]interface{}
	for rows.Next() {
		var sid int
		var name string
		rows.Scan(&sid, &name)

		gradeRows, err := s.db.Query(
			`SELECT EXTRACT(DAY FROM lesson_date)::int, grade, status
			 FROM ocenki
			 WHERE student_id=$1 AND subject_id=$2
			   AND lesson_date >= $3
			   AND lesson_date < ($3::date + INTERVAL '1 month')`,
			sid, subjectID, firstDay,
		)
		if err != nil {
			continue
		}
		grades := make(map[int]gradeInfo)
		for gradeRows.Next() {
			var day int
			var grade sql.NullInt64
			var status string
			gradeRows.Scan(&day, &grade, &status)
			var gp *int
			if grade.Valid {
				g := int(grade.Int64)
				gp = &g
			}
			grades[day] = gradeInfo{grade: gp, status: status}
		}
		gradeRows.Close()

		row := map[string]interface{}{
			"student_id":   sid,
			"student_name": name,
		}
		for d := 1; d <= daysCount; d++ {
			key := fmt.Sprintf("day_%d", d)
			if info, ok := grades[d]; ok {
				if info.grade != nil {
					row[key] = map[string]interface{}{"value": *info.grade, "status": info.status}
				} else {
					row[key] = map[string]interface{}{"value": nil, "status": info.status}
				}
			} else {
				row[key] = map[string]interface{}{"value": nil, "status": ""}
			}
		}
		result = append(result, row)
	}
	return result, nil
}

func (s *Storage) SetGrade(studentID, subjectID int, lessonDate string, grade int) error {
	_, err := s.db.Exec(
		`INSERT INTO ocenki (student_id, subject_id, lesson_date, grade, status)
		 VALUES ($1, $2, $3, $4, 'present')
		 ON CONFLICT (student_id, subject_id, lesson_date)
		 DO UPDATE SET grade=EXCLUDED.grade, status='present'`,
		studentID, subjectID, lessonDate, grade,
	)
	return err
}

func (s *Storage) SetAbsent(studentID, subjectID int, lessonDate, reason string) error {
	_, err := s.db.Exec(
		`INSERT INTO ocenki (student_id, subject_id, lesson_date, grade, status)
		 VALUES ($1, $2, $3, NULL, $4)
		 ON CONFLICT (student_id, subject_id, lesson_date)
		 DO UPDATE SET grade=NULL, status=EXCLUDED.status`,
		studentID, subjectID, lessonDate, reason,
	)
	return err
}

func (s *Storage) ClearGrade(studentID, subjectID int, lessonDate string) error {
	_, err := s.db.Exec(
		`DELETE FROM ocenki WHERE student_id=$1 AND subject_id=$2 AND lesson_date=$3`,
		studentID, subjectID, lessonDate,
	)
	return err
}

func (s *Storage) SeedData(groups map[string][]string, studentsPerGroup map[string][]string) error {
	for gname, subs := range groups {
		gid, err := s.AddGroup(gname)
		if err != nil {
			return fmt.Errorf("add group %s: %w", gname, err)
		}
		for _, sub := range subs {
			if _, err := s.AddSubject(sub, gid); err != nil {
				return fmt.Errorf("add subject %s: %w", sub, err)
			}
		}
		for _, stname := range studentsPerGroup[gname] {
			if _, err := s.AddStudent(stname, gid); err != nil {
				return fmt.Errorf("add student %s: %w", stname, err)
			}
		}
	}
	return nil
}
