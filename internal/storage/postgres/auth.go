package postgres

import (
	"database/sql"
	"errors"
	"fmt"
)

type Role string

const (
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
)

type User struct {
	ID        int
	FullName  string
	Role      Role
	GroupID   *int
	GroupName string
}

func CreateAuthTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id         SERIAL PRIMARY KEY,
			full_name  TEXT NOT NULL,
			role       TEXT NOT NULL CHECK (role IN ('teacher', 'student')),
			group_id   INTEGER REFERENCES groups(id) ON DELETE SET NULL,
			password   TEXT NOT NULL,
			UNIQUE (full_name, role, group_id)
		)
	`)
	return err
}

func (s *Storage) Login(fullName, password string, role Role, groupID *int) (*User, error) {
	query := `
		SELECT u.id, u.full_name, u.role, u.group_id, g.name
		FROM users u
		LEFT JOIN groups g ON g.id = u.group_id
		WHERE u.full_name = $1
		  AND u.password = $2
		  AND u.role = $3
	`
	args := []interface{}{fullName, password, string(role)}

	if role == RoleStudent && groupID != nil {
		query += " AND u.group_id = $4"
		args = append(args, *groupID)
	}

	row := s.db.QueryRow(query, args...)

	var u User
	var gid sql.NullInt64
	var gname sql.NullString
	err := row.Scan(&u.ID, &u.FullName, &u.Role, &gid, &gname)
	if err == sql.ErrNoRows {
		return nil, errors.New("неверные данные")
	}
	if err != nil {
		return nil, fmt.Errorf("login query: %w", err)
	}
	if gid.Valid {
		id := int(gid.Int64)
		u.GroupID = &id
	}
	if gname.Valid {
		u.GroupName = gname.String
	}
	return &u, nil
}

func (s *Storage) GetUserByID(id int) (*User, error) {
	row := s.db.QueryRow(`
		SELECT u.id, u.full_name, u.role, u.group_id, g.name
		FROM users u
		LEFT JOIN groups g ON g.id = u.group_id
		WHERE u.id = $1
	`, id)

	var u User
	var gid sql.NullInt64
	var gname sql.NullString
	err := row.Scan(&u.ID, &u.FullName, &u.Role, &gid, &gname)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if gid.Valid {
		id := int(gid.Int64)
		u.GroupID = &id
	}
	if gname.Valid {
		u.GroupName = gname.String
	}
	return &u, nil
}

func (s *Storage) SeedUsers() error {
	teachers := []string{"Иванова А.В.", "Петров С.И.", "Сидорова М.П."}
	for _, name := range teachers {
		_, err := s.db.Exec(`
			INSERT INTO users (full_name, role, group_id, password)
			VALUES ($1, 'teacher', NULL, '1234')
			ON CONFLICT (full_name, role, group_id) DO NOTHING
		`, name)
		if err != nil {
			return err
		}
	}

	rows, err := s.db.Query(`
		SELECT s.full_name, s.group_id
		FROM students s
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var gid int
		rows.Scan(&name, &gid)
		_, err := s.db.Exec(`
			INSERT INTO users (full_name, role, group_id, password)
			VALUES ($1, 'student', $2, '1234')
			ON CONFLICT (full_name, role, group_id) DO NOTHING
		`, name, gid)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) GetStudentGradesAllSubjects(studentID, groupID, year, month int) ([]map[string]interface{}, error) {
	subjects, err := s.GetSubjectsByGroup(groupID)
	if err != nil {
		return nil, err
	}

	firstDay := fmt.Sprintf("%d-%02d-01", year, month)

	var result []map[string]interface{}
	for _, subj := range subjects {
		sid := subj["id"].(int)
		sname := subj["name"].(string)

		rows, err := s.db.Query(`
			SELECT EXTRACT(DAY FROM lesson_date)::int, grade, status
			FROM ocenki
			WHERE student_id = $1 AND subject_id = $2
			  AND lesson_date >= $3
			  AND lesson_date < ($3::date + INTERVAL '1 month')
			ORDER BY lesson_date
		`, studentID, sid, firstDay)
		if err != nil {
			continue
		}

		var grades []map[string]interface{}
		var sum float64
		var cnt int
		var absTotal, absExcused int

		for rows.Next() {
			var day int
			var grade sql.NullInt64
			var status string
			rows.Scan(&day, &grade, &status)

			entry := map[string]interface{}{
				"day":    day,
				"status": status,
				"value":  nil,
			}
			if grade.Valid {
				entry["value"] = int(grade.Int64)
				sum += float64(grade.Int64)
				cnt++
			}
			if status == "absent" {
				absTotal += 2
			} else if status == "excused" {
				absTotal += 2
				absExcused += 2
			}
			grades = append(grades, entry)
		}
		rows.Close()

		var avg *float64
		if cnt > 0 {
			a := sum / float64(cnt)
			avg = &a
		}

		result = append(result, map[string]interface{}{
			"subject_id":   sid,
			"subject_name": sname,
			"grades":       grades,
			"avg":          avg,
			"abs_total":    absTotal,
			"abs_excused":  absExcused,
		})
	}
	return result, nil
}
