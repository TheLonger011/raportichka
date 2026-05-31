package postgres

import (
	"database/sql"
	"fmt"

	"github.com/TheLonger011/raportichka/internal/domain"
)

func (s *Storage) GetGrades(groupID, subjectID, year, month int) ([]domain.StudentGradeRow, error) {
	firstDay := fmt.Sprintf("%d-%02d-01", year, month)

	var daysCount int
	err := s.db.QueryRow(
		`SELECT EXTRACT(DAY FROM ($1::date + INTERVAL '1 month - 1 day'))::int`,
		firstDay,
	).Scan(&daysCount)
	if err != nil {
		return nil, err
	}

	students, err := s.GetStudentsByGroup(groupID)
	if err != nil {
		return nil, err
	}

	var result []domain.StudentGradeRow
	for _, st := range students {
		gradeRows, err := s.db.Query(
			`SELECT EXTRACT(DAY FROM lesson_date)::int, grade, status
			 FROM ocenki
			 WHERE student_id=$1 AND subject_id=$2
			   AND lesson_date >= $3
			   AND lesson_date < ($3::date + INTERVAL '1 month')`,
			st.ID, subjectID, firstDay,
		)
		if err != nil {
			continue
		}

		cells := make(map[string]domain.GradeCell, daysCount)
		for gradeRows.Next() {
			var day int
			var grade sql.NullInt64
			var status string
			gradeRows.Scan(&day, &grade, &status)

			cell := domain.GradeCell{Status: status}
			if grade.Valid {
				v := int(grade.Int64)
				cell.Value = &v
			}
			cells[fmt.Sprintf("day_%d", day)] = cell
		}
		gradeRows.Close()

		for d := 1; d <= daysCount; d++ {
			key := fmt.Sprintf("day_%d", d)
			if _, ok := cells[key]; !ok {
				cells[key] = domain.GradeCell{}
			}
		}

		result = append(result, domain.StudentGradeRow{
			StudentID:   st.ID,
			StudentName: st.FullName,
			Days:        cells,
		})
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

func (s *Storage) GetStudentGradesAllSubjects(studentID, groupID, year, month int) ([]domain.SubjectGrades, error) {
	subjects, err := s.GetSubjectsByGroup(groupID)
	if err != nil {
		return nil, err
	}

	firstDay := fmt.Sprintf("%d-%02d-01", year, month)

	var result []domain.SubjectGrades
	for _, subj := range subjects {
		rows, err := s.db.Query(`
			SELECT EXTRACT(DAY FROM lesson_date)::int, grade, status
			FROM ocenki
			WHERE student_id = $1 AND subject_id = $2
			  AND lesson_date >= $3
			  AND lesson_date < ($3::date + INTERVAL '1 month')
			ORDER BY lesson_date
		`, studentID, subj.ID, firstDay)
		if err != nil {
			continue
		}

		var entries []domain.GradeEntry
		var sum float64
		var cnt int
		var absTotal, absExcused int

		for rows.Next() {
			var day int
			var grade sql.NullInt64
			var status string
			rows.Scan(&day, &grade, &status)

			entry := domain.GradeEntry{Day: day, Status: status}
			if grade.Valid {
				v := int(grade.Int64)
				entry.Value = &v
				sum += float64(v)
				cnt++
			}
			if status == "absent" {
				absTotal += 2
			} else if status == "excused" {
				absTotal += 2
				absExcused += 2
			}
			entries = append(entries, entry)
		}
		rows.Close()

		sg := domain.SubjectGrades{
			SubjectID:   subj.ID,
			SubjectName: subj.Name,
			Grades:      entries,
			AbsTotal:    absTotal,
			AbsExcused:  absExcused,
		}
		if cnt > 0 {
			avg := sum / float64(cnt)
			sg.Avg = &avg
		}
		result = append(result, sg)
	}
	return result, nil
}

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
