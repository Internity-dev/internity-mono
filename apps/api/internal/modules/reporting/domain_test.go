package reporting

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildStudentRosterExcel(t *testing.T) {
	rows := []StudentRosterRow{
		{Name: "Budi Santoso", NIS: "1001", CourseName: "XII RPL 1"},
		{Name: "Siti Aminah", NIS: "1002", CourseName: "XII RPL 2"},
	}

	f, err := BuildStudentRosterExcel(rows)
	require.NoError(t, err)
	require.NotNil(t, f)

	header, err := f.GetCellValue("Students", "A1")
	require.NoError(t, err)
	assert.Equal(t, "Name", header)

	name, err := f.GetCellValue("Students", "A2")
	require.NoError(t, err)
	assert.Equal(t, "Budi Santoso", name)

	nis, err := f.GetCellValue("Students", "B3")
	require.NoError(t, err)
	assert.Equal(t, "1002", nis)

	buf, err := f.WriteToBuffer()
	require.NoError(t, err)
	assert.Greater(t, buf.Len(), 0)
}

func TestBuildStudentRosterExcel_Empty(t *testing.T) {
	f, err := BuildStudentRosterExcel(nil)
	require.NoError(t, err)
	header, err := f.GetCellValue("Students", "A1")
	require.NoError(t, err)
	assert.Equal(t, "Name", header, "headers should still render with zero data rows")
}

func TestBuildPresenceExcel(t *testing.T) {
	checkIn := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	rows := []PresenceExportRow{
		{Date: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), CheckInAt: &checkIn, StatusName: "Hadir", IsApproved: true},
		{Date: time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC), StatusName: "Sakit", IsApproved: false, Description: "Flu"},
	}

	f, err := BuildPresenceExcel("Budi Santoso", "PT Contoh", rows)
	require.NoError(t, err)

	student, err := f.GetCellValue("Presence", "B1")
	require.NoError(t, err)
	assert.Equal(t, "Budi Santoso", student)

	status, err := f.GetCellValue("Presence", "D6")
	require.NoError(t, err)
	assert.Equal(t, "Sakit", status)
}
