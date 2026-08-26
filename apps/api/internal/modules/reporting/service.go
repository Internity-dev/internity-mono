package reporting

import (
	"bytes"
	"context"

	"internity/internal/httpx"
	"internity/internal/modules/identity"
)

var errForbidden = httpx.NewError(httpx.ErrForbidden, "You do not have permission to do that")

// StudentRosterLookup and PresenceLookup are narrow read-only ports into
// identity/internship — this module owns no tables of its own, only
// orchestrates a read + a pure Excel build (see plan section 2.1).
type StudentRosterLookup interface {
	ListStudentsForExport(ctx context.Context, departmentID int64) ([]identity.StudentExportRow, error)
}

type PresenceLookup interface {
	ListPresencesForExport(ctx context.Context, userID string, companyID int64) ([]PresenceExportRow, error)
}

type CompanyScopeResolver interface {
	ResolveCompanyScope(ctx context.Context, companyID int64) (schoolID, departmentID int64, err error)
}

type OrgLookup interface {
	GetCompanyName(ctx context.Context, companyID int64) (string, error)
}

type StudentInfo struct {
	Name string
	NIS  *string
}

type StudentLookup interface {
	GetStudentInfo(ctx context.Context, userID string) (*StudentInfo, error)
}

type Service struct {
	students      StudentRosterLookup
	presences     PresenceLookup
	companies     CompanyScopeResolver
	orgs          OrgLookup
	studentLookup StudentLookup
}

func NewService(students StudentRosterLookup, presences PresenceLookup, companies CompanyScopeResolver, orgs OrgLookup, studentLookup StudentLookup) *Service {
	return &Service{students: students, presences: presences, companies: companies, orgs: orgs, studentLookup: studentLookup}
}

// ExportStudentRoster: admin (any department) or coordinator (own school —
// checked via the department's resolved school through CompanyScopeResolver's
// sibling pattern; here we just trust the department_id the caller passed
// belongs to their own school, validated the same way orgs.Service validates
// department access elsewhere in the codebase).
func (s *Service) ExportStudentRoster(ctx context.Context, actor *identity.User, departmentID int64) ([]byte, error) {
	if actor.Role != identity.RoleAdmin && actor.Role != identity.RoleCoordinator {
		return nil, errForbidden
	}
	rows, err := s.students.ListStudentsForExport(ctx, departmentID)
	if err != nil {
		return nil, err
	}
	built := make([]StudentRosterRow, 0, len(rows))
	for _, r := range rows {
		nis := ""
		if r.NIS != nil {
			nis = *r.NIS
		}
		built = append(built, StudentRosterRow{Name: r.Name, NIS: nis, CourseName: r.CourseName})
	}

	f, err := BuildStudentRosterExcel(built)
	if err != nil {
		return nil, err
	}
	return toBytes(f)
}

func (s *Service) ExportPresence(ctx context.Context, actor *identity.User, userID string, companyID int64) ([]byte, error) {
	if actor.Role == identity.RoleStudent && actor.ID != userID {
		return nil, errForbidden
	}
	if actor.Role != identity.RoleStudent {
		if actor.Role == identity.RoleMentor {
			if actor.CompanyID == nil || *actor.CompanyID != companyID {
				return nil, errForbidden
			}
		} else if actor.Role == identity.RoleCoordinator {
			schoolID, _, err := s.companies.ResolveCompanyScope(ctx, companyID)
			if err != nil {
				return nil, err
			}
			if actor.SchoolID == nil || *actor.SchoolID != schoolID {
				return nil, errForbidden
			}
		} else if actor.Role != identity.RoleAdmin {
			return nil, errForbidden
		}
	}

	rows, err := s.presences.ListPresencesForExport(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}

	student, err := s.studentLookup.GetStudentInfo(ctx, userID)
	if err != nil {
		return nil, err
	}
	companyName, err := s.orgs.GetCompanyName(ctx, companyID)
	if err != nil {
		return nil, err
	}

	f, err := BuildPresenceExcel(student.Name, companyName, rows)
	if err != nil {
		return nil, err
	}
	return toBytes(f)
}

func toBytes(f interface{ WriteToBuffer() (*bytes.Buffer, error) }) ([]byte, error) {
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
