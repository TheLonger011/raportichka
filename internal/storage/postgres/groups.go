package postgres

import (
	"fmt"

	"github.com/TheLonger011/raportichka/internal/domain"
)

func (s *Storage) GetGroups() ([]domain.Group, error) {
	rows, err := s.db.Query(`SELECT id, name FROM groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Group
	for rows.Next() {
		var g domain.Group
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			return nil, err
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

func (s *Storage) AddGroup(name string) (int, error) {
	var id int
	err := s.db.QueryRow(
		`INSERT INTO groups (name) VALUES ($1)
		 ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name RETURNING id`,
		name,
	).Scan(&id)
	return id, err
}

func (s *Storage) GetSubjectsByGroup(groupID int) ([]domain.Subject, error) {
	rows, err := s.db.Query(
		`SELECT id, name, group_id FROM subjects WHERE group_id=$1 ORDER BY name`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Subject
	for rows.Next() {
		var sub domain.Subject
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.GroupID); err != nil {
			return nil, err
		}
		result = append(result, sub)
	}
	return result, rows.Err()
}

func (s *Storage) AddSubject(name string, groupID int) (int, error) {
	var id int
	err := s.db.QueryRow(
		`INSERT INTO subjects (name, group_id) VALUES ($1, $2)
		 ON CONFLICT (name, group_id) DO UPDATE SET name=EXCLUDED.name RETURNING id`,
		name, groupID,
	).Scan(&id)
	return id, err
}

func (s *Storage) GetStudentsByGroup(groupID int) ([]domain.Student, error) {
	rows, err := s.db.Query(
		`SELECT id, full_name, group_id FROM students WHERE group_id=$1 ORDER BY id`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Student
	for rows.Next() {
		var st domain.Student
		if err := rows.Scan(&st.ID, &st.FullName, &st.GroupID); err != nil {
			return nil, err
		}
		result = append(result, st)
	}
	return result, rows.Err()
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

func (s *Storage) DeleteStudent(studentID int) error {
	_, err := s.db.Exec(`DELETE FROM students WHERE id = $1`, studentID)
	return err
}

func (s *Storage) IsEmpty() (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM groups`).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}
