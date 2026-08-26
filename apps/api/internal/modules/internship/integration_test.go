//go:build integration

// Run with `make test-integration` (needs a local Docker daemon). This isn't
// exercised in CI/dev environments without Docker, so plain `go test ./...`
// never touches it — see the build tag above.
package internship

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// migrationsDir resolves apps/api/migrations relative to this file, not the
// test binary's working directory, so `go test -tags=integration ./...` from
// any directory finds the same migration set the app itself runs.
func migrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "migrations")
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("internity_test"),
		postgres.WithUsername("internity"),
		postgres.WithPassword("internity"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	m, err := migrate.New("file://"+filepath.ToSlash(migrationsDir()), dsn)
	require.NoError(t, err)
	require.NoError(t, m.Up())

	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	return db
}

// TestInternDates_NoOverlapPerUser verifies excl_intern_dates_no_overlap_per_user
// (migrations/000013_create_intern_dates.up.sql): a student's placement date
// ranges may not overlap, even across two different companies. This is
// DB-enforced (a GiST EXCLUDE constraint), not application logic, so a faked
// repository can never verify it — it needs a real Postgres.
func TestInternDates_NoOverlapPerUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	schoolID := seedSchool(t, db)
	departmentID := seedDepartment(t, db, schoolID)
	courseID := seedCourse(t, db, departmentID)
	userID := seedUser(t, db, schoolID, departmentID, courseID)
	companyA := seedCompany(t, db, departmentID, "Company A")
	companyB := seedCompany(t, db, departmentID, "Company B")
	applianceA := seedAppliance(t, db, userID, seedVacancy(t, db, companyA))
	applianceB := seedAppliance(t, db, userID, seedVacancy(t, db, companyB))

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	first := &InternDate{
		UserID: userID, CompanyID: companyA, ApplianceID: applianceA,
		StartDate: &start, EndDate: &end,
	}
	require.NoError(t, repo.Create(ctx, first))

	overlapStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	overlapEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	overlapping := &InternDate{
		UserID: userID, CompanyID: companyB, ApplianceID: applianceB,
		StartDate: &overlapStart, EndDate: &overlapEnd,
	}
	err := repo.Create(ctx, overlapping)
	require.Error(t, err, "the GiST exclusion constraint should reject an overlapping date range for the same user")

	nonOverlapStart := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	nonOverlapEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	sequential := &InternDate{
		UserID: userID, CompanyID: companyB, ApplianceID: applianceB,
		StartDate: &nonOverlapStart, EndDate: &nonOverlapEnd,
	}
	require.NoError(t, repo.Create(ctx, sequential), "a date range starting after the first one ends should be accepted")
}

func seedSchool(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var id int64
	require.NoError(t, db.Raw(`INSERT INTO schools (name) VALUES ('Test School') RETURNING id`).Scan(&id).Error)
	return id
}

func seedDepartment(t *testing.T, db *gorm.DB, schoolID int64) int64 {
	t.Helper()
	var id int64
	require.NoError(t, db.Raw(
		`INSERT INTO departments (school_id, name) VALUES (?, 'Test Department') RETURNING id`, schoolID,
	).Scan(&id).Error)
	return id
}

func seedCourse(t *testing.T, db *gorm.DB, departmentID int64) int64 {
	t.Helper()
	var id int64
	require.NoError(t, db.Raw(
		`INSERT INTO courses (department_id, name) VALUES (?, 'Test Course') RETURNING id`, departmentID,
	).Scan(&id).Error)
	return id
}

func seedUser(t *testing.T, db *gorm.DB, schoolID, departmentID, courseID int64) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, role, name, email, password_hash, is_active, school_id, department_id, course_id)
		 VALUES (?, 'student', 'Test Student', ?, 'x', true, ?, ?, ?)`,
		id, id+"@example.com", schoolID, departmentID, courseID,
	).Error)
	return id
}

func seedCompany(t *testing.T, db *gorm.DB, departmentID int64, name string) int64 {
	t.Helper()
	var id int64
	require.NoError(t, db.Raw(
		`INSERT INTO companies (department_id, name) VALUES (?, ?) RETURNING id`, departmentID, name,
	).Scan(&id).Error)
	return id
}

func seedVacancy(t *testing.T, db *gorm.DB, companyID int64) int64 {
	t.Helper()
	var id int64
	require.NoError(t, db.Raw(
		`INSERT INTO vacancies (company_id, name) VALUES (?, 'Test Vacancy') RETURNING id`, companyID,
	).Scan(&id).Error)
	return id
}

func seedAppliance(t *testing.T, db *gorm.DB, userID string, vacancyID int64) int64 {
	t.Helper()
	var id int64
	require.NoError(t, db.Raw(
		`INSERT INTO appliances (user_id, vacancy_id, status) VALUES (?, ?, 'accepted') RETURNING id`,
		userID, vacancyID,
	).Scan(&id).Error)
	return id
}
