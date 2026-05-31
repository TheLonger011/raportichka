package vedomost

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"

	"github.com/TheLonger011/raportichka/internal/domain"
	"github.com/TheLonger011/raportichka/internal/storage/postgres"
)

const (
	TemplatePath = "vedomost/template.xlsx"
	ScriptPath   = "vedomost/gen_vedomost.py"
)

var monthNames = [13]string{
	"", "январь", "февраль", "март", "апрель", "май", "июнь",
	"июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь",
}

type Result struct {
	Filename string
	Data     []byte
}

type Service struct {
	storage *postgres.Storage
}

func NewService(storage *postgres.Storage) *Service {
	return &Service{storage: storage}
}

func (s *Service) Generate(
	ctx context.Context,
	groupID, year, month int,
	subjectIDs []int,
	totalHours int,
	headTeacher, deptHead string,
) (*Result, error) {
	groups, err := s.storage.GetGroups()
	if err != nil {
		return nil, err
	}
	groupName := groupNameByID(groups, groupID)

	allSubjects, err := s.storage.GetSubjectsByGroup(groupID)
	if err != nil {
		return nil, err
	}

	subjects := filterSubjects(allSubjects, subjectIDs)
	if len(subjects) == 0 {
		return nil, fmt.Errorf("no subjects selected")
	}

	studentsDB, err := s.storage.GetStudentsByGroup(groupID)
	if err != nil {
		return nil, err
	}

	students, err := s.aggregateStudents(studentsDB, subjects, groupID, year, month)
	if err != nil {
		return nil, err
	}

	mLabel := buildMonthLabel(month, year)

	tmpFile, err := os.CreateTemp("", "vedomost_*.xlsx")
	if err != nil {
		return nil, fmt.Errorf("tmp file: %w", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	payload := buildPayload(groupName, mLabel, subjects, students, totalHours, headTeacher, deptHead, tmpFile.Name())
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("json marshal: %w", err)
	}

	cmd := exec.CommandContext(ctx, "python3", ScriptPath)
	cmd.Stdin = strings.NewReader(string(payloadJSON))
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("gen script: %w\n%s", err, string(out))
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("read output: %w", err)
	}

	safeGroup := strings.ReplaceAll(groupName, " ", "_")
	filename := fmt.Sprintf("vedomost_%s_%d_%02d.xlsx", safeGroup, year, month)

	return &Result{Filename: filename, Data: data}, nil
}

type pySubject struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type pyStudent struct {
	FullName        string         `json:"full_name"`
	Grades          map[string]int `json:"grades"`
	AbsentTotal     int            `json:"absent_total"`
	AbsentExcused   int            `json:"absent_excused"`
	AbsentUnexcused int            `json:"absent_unexcused"`
}

type pyPayload struct {
	GroupName    string      `json:"group_name"`
	MonthLabel   string      `json:"month_label"`
	Subjects     []pySubject `json:"subjects"`
	Students     []pyStudent `json:"students"`
	TotalHours   int         `json:"total_hours"`
	HeadTeacher  string      `json:"head_teacher"`
	DeptHead     string      `json:"dept_head"`
	TemplatePath string      `json:"template_path"`
	OutputPath   string      `json:"output_path"`
}

func groupNameByID(groups []domain.Group, id int) string {
	for _, g := range groups {
		if g.ID == id {
			return g.Name
		}
	}
	return ""
}

func filterSubjects(all []domain.Subject, ids []int) []domain.Subject {
	set := make(map[int]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	var result []domain.Subject
	for _, s := range all {
		if set[s.ID] {
			result = append(result, s)
		}
		if len(result) >= 6 {
			break
		}
	}
	return result
}

type gradeKey struct{ studentID, subjectID int }
type cellData struct {
	grades  []int
	absents int
	excused int
}

func (s *Service) aggregateStudents(
	studentsDB []domain.Student,
	subjects []domain.Subject,
	groupID, year, month int,
) ([]pyStudent, error) {
	allCells := make(map[gradeKey]*cellData)

	for _, subj := range subjects {
		rows, err := s.storage.GetGrades(groupID, subj.ID, year, month)
		if err != nil {
			continue
		}
		for _, row := range rows {
			key := gradeKey{row.StudentID, subj.ID}
			if allCells[key] == nil {
				allCells[key] = &cellData{}
			}
			cd := allCells[key]
			for _, cell := range row.Days {
				switch cell.Status {
				case "absent":
					cd.absents++
				case "excused":
					cd.excused++
				default:
					if cell.Value != nil {
						cd.grades = append(cd.grades, *cell.Value)
					}
				}
			}
		}
	}

	var students []pyStudent
	for _, st := range studentsDB {
		grades := make(map[string]int)
		totalAbsent := 0
		totalExcused := 0

		for _, subj := range subjects {
			key := gradeKey{st.ID, subj.ID}
			cd := allCells[key]
			if cd == nil {
				continue
			}
			totalAbsent += cd.absents
			totalExcused += cd.excused

			if len(cd.grades) > 0 {
				sum := 0
				for _, g := range cd.grades {
					sum += g
				}
				avg := float64(sum) / float64(len(cd.grades))
				rounded := int(math.Floor(avg + 0.5))
				if rounded >= 2 && rounded <= 5 {
					grades[fmt.Sprintf("%d", subj.ID)] = rounded
				}
			}
		}

		students = append(students, pyStudent{
			FullName:        st.FullName,
			Grades:          grades,
			AbsentTotal:     totalAbsent + totalExcused,
			AbsentExcused:   totalExcused,
			AbsentUnexcused: totalAbsent,
		})
	}
	return students, nil
}

func buildMonthLabel(month, year int) string {
	if month < 1 || month > 12 {
		return ""
	}
	y1, y2 := year-1, year
	if month >= 9 {
		y1, y2 = year, year+1
	}
	return fmt.Sprintf("%s %d-%d уч.г.", monthNames[month], y1, y2)
}

func buildPayload(
	groupName, mLabel string,
	subjects []domain.Subject,
	students []pyStudent,
	totalHours int,
	headTeacher, deptHead, outputPath string,
) pyPayload {
	pySubs := make([]pySubject, len(subjects))
	for i, s := range subjects {
		pySubs[i] = pySubject{ID: s.ID, Name: s.Name}
	}
	return pyPayload{
		GroupName:    groupName,
		MonthLabel:   mLabel,
		Subjects:     pySubs,
		Students:     students,
		TotalHours:   totalHours,
		HeadTeacher:  headTeacher,
		DeptHead:     deptHead,
		TemplatePath: TemplatePath,
		OutputPath:   outputPath,
	}
}
