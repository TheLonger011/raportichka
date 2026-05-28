package vedomost

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"raportichka/internal/storage/postgres"
	"strings"
	"time"
)

const (
	vedomostTemplatePath = "vedomost/template.xlsx"
	vedomostScriptPath   = "vedomost/gen_vedomost.py"
)

type VedomostRequest struct {
	GroupID     int    `json:"group_id"`
	Year        int    `json:"year"`
	Month       int    `json:"month"`
	SubjectIDs  []int  `json:"subject_ids"`
	TotalHours  int    `json:"total_hours"`
	HeadTeacher string `json:"head_teacher"`
	DeptHead    string `json:"dept_head"`
}

type vedomostSubject struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type vedomostStudent struct {
	FullName        string         `json:"full_name"`
	Grades          map[string]int `json:"grades"`
	AbsentTotal     int            `json:"absent_total"`
	AbsentExcused   int            `json:"absent_excused"`
	AbsentUnexcused int            `json:"absent_unexcused"`
}

type vedomostPayload struct {
	GroupName    string            `json:"group_name"`
	MonthLabel   string            `json:"month_label"`
	Subjects     []vedomostSubject `json:"subjects"`
	Students     []vedomostStudent `json:"students"`
	TotalHours   int               `json:"total_hours"`
	HeadTeacher  string            `json:"head_teacher"`
	DeptHead     string            `json:"dept_head"`
	TemplatePath string            `json:"template_path"`
	OutputPath   string            `json:"output_path"`
}

var monthNames = [13]string{
	"", "январь", "февраль", "март", "апрель", "май", "июнь",
	"июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь",
}

func roundGrade(avg float64) int {
	return int(math.Floor(avg + 0.5))
}

func MakeHandleVedomost(storage *postgres.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}

		var req VedomostRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request: "+err.Error(), 400)
			return
		}
		if req.Year == 0 {
			req.Year = 2026
		}
		if req.Month == 0 {
			req.Month = 5
		}
		if req.TotalHours == 0 {
			req.TotalHours = 120
		}

		groups, err := storage.GetGroups()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		groupName := ""
		for _, g := range groups {
			if id, ok := g["id"].(int); ok && id == req.GroupID {
				groupName, _ = g["name"].(string)
				break
			}
		}

		allSubjects, err := storage.GetSubjectsByGroup(req.GroupID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		subjectIDSet := make(map[int]bool)
		for _, id := range req.SubjectIDs {
			subjectIDSet[id] = true
		}
		var subjects []vedomostSubject
		for _, s := range allSubjects {
			id, _ := s["id"].(int)
			name, _ := s["name"].(string)
			if subjectIDSet[id] {
				subjects = append(subjects, vedomostSubject{ID: id, Name: name})
			}
			if len(subjects) >= 6 {
				break
			}
		}

		studentsDB, err := storage.GetStudentsByGroup(req.GroupID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		type gradeKey struct{ studentID, subjectID int }
		type cellData struct {
			grades  []int
			absents int
			excused int
		}
		allCells := make(map[gradeKey]*cellData)

		for _, subj := range subjects {
			rows, err := storage.GetGrades(req.GroupID, subj.ID, req.Year, req.Month)
			if err != nil {
				continue
			}
			for _, row := range rows {
				sid, _ := row["student_id"].(int)
				key := gradeKey{sid, subj.ID}
				if allCells[key] == nil {
					allCells[key] = &cellData{}
				}
				cd := allCells[key]
				for k, v := range row {
					if !strings.HasPrefix(k, "day_") {
						continue
					}
					dayMap, ok := v.(map[string]interface{})
					if !ok {
						continue
					}
					status, _ := dayMap["status"].(string)
					val := dayMap["value"]
					if status == "absent" {
						cd.absents++
					} else if status == "excused" {
						cd.excused++
					} else if val != nil {
						switch g := val.(type) {
						case int:
							cd.grades = append(cd.grades, g)
						case float64:
							cd.grades = append(cd.grades, int(g))
						}
					}
				}
			}
		}

		var students []vedomostStudent
		for _, st := range studentsDB {
			stID, _ := st["id"].(int)
			name, _ := st["full_name"].(string)

			grades := make(map[string]int)
			totalAbsent := 0
			totalExcused := 0

			for _, subj := range subjects {
				key := gradeKey{stID, subj.ID}
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
					rounded := roundGrade(avg)
					if rounded >= 2 && rounded <= 5 {
						grades[fmt.Sprintf("%d", subj.ID)] = rounded
					}
				}
			}

			students = append(students, vedomostStudent{
				FullName:        name,
				Grades:          grades,
				AbsentTotal:     totalAbsent + totalExcused,
				AbsentExcused:   totalExcused,
				AbsentUnexcused: totalAbsent,
			})
		}

		mLabel := ""
		if req.Month >= 1 && req.Month <= 12 {
			y1, y2 := req.Year-1, req.Year
			if req.Month >= 9 {
				y1, y2 = req.Year, req.Year+1
			}
			mLabel = fmt.Sprintf("%s %d-%d уч.г.", monthNames[req.Month], y1, y2)
		}

		tmpFile, err := os.CreateTemp("", "vedomost_*.xlsx")
		if err != nil {
			http.Error(w, "cannot create tmp file: "+err.Error(), 500)
			return
		}
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		payload := vedomostPayload{
			GroupName:    groupName,
			MonthLabel:   mLabel,
			Subjects:     subjects,
			Students:     students,
			TotalHours:   req.TotalHours,
			HeadTeacher:  req.HeadTeacher,
			DeptHead:     req.DeptHead,
			TemplatePath: vedomostTemplatePath,
			OutputPath:   tmpFile.Name(),
		}

		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, "json marshal: "+err.Error(), 500)
			return
		}

		cmd := exec.Command("python3", vedomostScriptPath)
		cmd.Stdin = strings.NewReader(string(payloadJSON))
		out, err := cmd.CombinedOutput()
		if err != nil {
			http.Error(w, fmt.Sprintf("gen script error: %v\n%s", err, string(out)), 500)
			return
		}

		xlsxBytes, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			http.Error(w, "cannot read output: "+err.Error(), 500)
			return
		}

		safeGroup := strings.ReplaceAll(groupName, " ", "_")
		filename := fmt.Sprintf("vedomost_%s_%d_%02d.xlsx", safeGroup, req.Year, req.Month)
		_ = time.Now()

		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(xlsxBytes)))
		w.Write(xlsxBytes)
	}
}
