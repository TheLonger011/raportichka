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
	createTableStudents := `
	CREATE TABLE IF NOT EXISTS students (
		id SERIAL PRIMARY KEY,
		full_name TEXT NOT NULL UNIQUE
	)`

	if _, err := db.Exec(createTableStudents); err != nil {
		return fmt.Errorf("failed to create students table: %w", err)
	}

	createTableOcenki := `
	CREATE TABLE IF NOT EXISTS ocenki (
		id SERIAL PRIMARY KEY,
		student_id INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE,
		lesson_date DATE NOT NULL,
		grade INT CHECK (grade >= 2 AND grade <= 5),
		status TEXT DEFAULT 'present',
		UNIQUE (student_id, lesson_date)
	)`

	if _, err := db.Exec(createTableOcenki); err != nil {
		return fmt.Errorf("failed to create ocenki table: %w", err)
	}

	return nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) AddStudent(fullName string) (int, error) {
	const op = "storage.postgres.AddStudent"

	var id int
	query := `INSERT INTO students (full_name) VALUES ($1) ON CONFLICT (full_name) DO NOTHING RETURNING id`
	err := s.db.QueryRow(query, fullName).Scan(&id)
	if err != nil {
		query = `SELECT id FROM students WHERE full_name = $1`
		err = s.db.QueryRow(query, fullName).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", op, err)
		}
	}
	return id, nil
}

func (s *Storage) LoadStudentsFromFile(students []string) error {
	for _, name := range students {
		_, err := s.AddStudent(name)
		if err != nil {
			return fmt.Errorf("failed to add student %s: %w", name, err)
		}
	}
	return nil
}

func (s *Storage) SetGrade(studentID int, lessonDate string, grade int) error {
	const op = "storage.postgres.SetGrade"

	query := `
		INSERT INTO ocenki (student_id, lesson_date, grade, status)
		VALUES ($1, $2, $3, 'present')
		ON CONFLICT (student_id, lesson_date)
		DO UPDATE SET grade = EXCLUDED.grade, status = 'present'
	`
	_, err := s.db.Exec(query, studentID, lessonDate, grade)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *Storage) SetAbsent(studentID int, lessonDate string, reason string) error {
	const op = "storage.postgres.SetAbsent"

	query := `
		INSERT INTO ocenki (student_id, lesson_date, grade, status)
		VALUES ($1, $2, NULL, $3)
		ON CONFLICT (student_id, lesson_date)
		DO UPDATE SET grade = NULL, status = EXCLUDED.status
	`
	_, err := s.db.Exec(query, studentID, lessonDate, reason)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *Storage) GetGrade(year int, month int) ([]map[string]interface{}, error) {
	const op = "storage.postgres.GetGrade"

	firstDay := fmt.Sprintf("%d-%02d-01", year, month)

	var daysCount int
	queryDays := `
		SELECT EXTRACT(DAY FROM ($1::date + INTERVAL '1 month - 1 day'))::int
	`
	err := s.db.QueryRow(queryDays, firstDay).Scan(&daysCount)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	studentsQuery := `
		SELECT id, full_name
		FROM students
		ORDER BY id ASC
	`

	rows, err := s.db.Query(studentsQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	result := []map[string]interface{}{}

	for rows.Next() {
		var id int
		var fullName string
		err := rows.Scan(&id, &fullName)
		if err != nil {
			continue
		}

		gradesQuery := `
			SELECT EXTRACT(DAY FROM lesson_date)::int as day, grade, status
			FROM ocenki
			WHERE student_id = $1
			AND lesson_date >= $2
			AND lesson_date < ($2::date + INTERVAL '1 month')
		`

		gradeRows, err := s.db.Query(gradesQuery, id, firstDay)
		if err != nil {
			continue
		}

		type gradeInfo struct {
			grade  *int
			status string
		}
		grades := make(map[int]gradeInfo)

		for gradeRows.Next() {
			var day int
			var grade sql.NullInt64
			var status string
			gradeRows.Scan(&day, &grade, &status)

			var gradePtr *int
			if grade.Valid {
				g := int(grade.Int64)
				gradePtr = &g
			}

			grades[day] = gradeInfo{
				grade:  gradePtr,
				status: status,
			}
		}
		gradeRows.Close()

		studentSheet := map[string]interface{}{
			"student_id":   id,
			"student_name": fullName,
		}

		for day := 1; day <= daysCount; day++ {
			key := fmt.Sprintf("day_%d", day)
			if info, exists := grades[day]; exists {
				if info.grade != nil {
					studentSheet[key] = map[string]interface{}{
						"value":  *info.grade,
						"status": info.status,
					}
				} else {
					studentSheet[key] = map[string]interface{}{
						"value":  nil,
						"status": info.status,
					}
				}
			} else {
				studentSheet[key] = map[string]interface{}{
					"value":  nil,
					"status": "",
				}
			}
		}
		result = append(result, studentSheet)
	}

	return result, nil
}
