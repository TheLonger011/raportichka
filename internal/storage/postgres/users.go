package postgres

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/TheLonger011/raportichka/internal/domain"
)

func (s *Storage) Login(fullName, password string, role domain.Role, groupID *int) (*domain.User, error) {
	query := `
		SELECT u.id, u.full_name, u.role, u.group_id, g.name
		FROM users u
		LEFT JOIN groups g ON g.id = u.group_id
		WHERE u.full_name = $1
		  AND u.password = $2
		  AND u.role = $3
	`
	args := []interface{}{fullName, password, string(role)}

	if role == domain.RoleStudent && groupID != nil {
		query += " AND u.group_id = $4"
		args = append(args, *groupID)
	}

	row := s.db.QueryRow(query, args...)

	var u domain.User
	var gid sql.NullInt64
	var gname sql.NullString
	err := row.Scan(&u.ID, &u.FullName, &u.Role, &gid, &gname)
	if errors.Is(err, sql.ErrNoRows) {
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

func (s *Storage) GetUserByID(id int) (*domain.User, error) {
	row := s.db.QueryRow(`
		SELECT u.id, u.full_name, u.role, u.group_id, g.name
		FROM users u
		LEFT JOIN groups g ON g.id = u.group_id
		WHERE u.id = $1
	`, id)

	var u domain.User
	var gid sql.NullInt64
	var gname sql.NullString
	err := row.Scan(&u.ID, &u.FullName, &u.Role, &gid, &gname)
	if errors.Is(err, sql.ErrNoRows) {
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

	rows, err := s.db.Query(`SELECT s.full_name, s.group_id FROM students s`)
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
	return rows.Err()
}
