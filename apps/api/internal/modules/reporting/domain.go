// Package reporting orchestrates Excel exports — it owns no tables of its
// own, only pure builder functions over data read (read-only) from other
// modules via the narrow lookup interfaces in service.go.
package reporting

import (
	"time"

	"github.com/xuri/excelize/v2"
)

type StudentRosterRow struct {
	Name       string
	NIS        string
	CourseName string
}

// BuildStudentRosterExcel is a pure function over already-fetched rows — no
// DB/IO here, which is what makes it unit-testable without a database.
func BuildStudentRosterExcel(rows []StudentRosterRow) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Students"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}

	headers := []string{"Name", "NIS", "Class"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	for i, r := range rows {
		row := i + 2
		a, _ := excelize.CoordinatesToCellName(1, row)
		b, _ := excelize.CoordinatesToCellName(2, row)
		c, _ := excelize.CoordinatesToCellName(3, row)
		f.SetCellValue(sheet, a, r.Name)
		f.SetCellValue(sheet, b, r.NIS)
		f.SetCellValue(sheet, c, r.CourseName)
	}
	return f, nil
}

type PresenceExportRow struct {
	Date        time.Time
	CheckInAt   *time.Time
	CheckOutAt  *time.Time
	StatusName  string
	IsApproved  bool
	Description string
}

func BuildPresenceExcel(studentName, companyName string, rows []PresenceExportRow) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Presence"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}

	f.SetCellValue(sheet, "A1", "Student")
	f.SetCellValue(sheet, "B1", studentName)
	f.SetCellValue(sheet, "A2", "Company")
	f.SetCellValue(sheet, "B2", companyName)

	headerRow := 4
	headers := []string{"Date", "Check In", "Check Out", "Status", "Approved", "Description"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRow)
		f.SetCellValue(sheet, cell, h)
	}
	for i, r := range rows {
		row := headerRow + 1 + i
		values := []any{
			r.Date.Format("2006-01-02"),
			formatTimeOrDash(r.CheckInAt), formatTimeOrDash(r.CheckOutAt),
			r.StatusName, formatBool(r.IsApproved), r.Description,
		}
		for col, v := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			f.SetCellValue(sheet, cell, v)
		}
	}
	return f, nil
}

func formatTimeOrDash(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("15:04")
}

func formatBool(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
