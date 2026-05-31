package domain

import "time"

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

type Group struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Subject struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	GroupID int    `json:"group_id"`
}

type Student struct {
	ID       int    `json:"id"`
	FullName string `json:"full_name"`
	GroupID  int    `json:"group_id"`
}

type GradeCell struct {
	Value  *int   `json:"value"`
	Status string `json:"status"`
}

type StudentGradeRow struct {
	StudentID   int                  `json:"student_id"`
	StudentName string               `json:"student_name"`
	Days        map[string]GradeCell `json:"days"`
}

type SubjectGrades struct {
	SubjectID   int          `json:"subject_id"`
	SubjectName string       `json:"subject_name"`
	Grades      []GradeEntry `json:"grades"`
	Avg         *float64     `json:"avg"`
	AbsTotal    int          `json:"abs_total"`
	AbsExcused  int          `json:"abs_excused"`
}

type GradeEntry struct {
	Day    int    `json:"day"`
	Status string `json:"status"`
	Value  *int   `json:"value"`
}

type FileInfo struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	Path     string    `json:"path"`
	Type     string    `json:"type"`
}
