package postgres

import "fmt"

func (s *Storage) GetStudentIDByUserID(userID int) (int, error) {
	var studentID int
	err := s.db.QueryRow(`
		SELECT st.id
		FROM students st
		JOIN users u ON u.full_name = st.full_name AND u.group_id = st.group_id
		WHERE u.id = $1
	`, userID).Scan(&studentID)
	if err != nil {
		return 0, fmt.Errorf("GetStudentIDByUserID: %w", err)
	}
	return studentID, nil
}
