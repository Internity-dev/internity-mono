package scoring

import (
	"context"
	"errors"
	"fmt"
	"time"

	"internity/internal/httpx"
	"internity/internal/modules/identity"
	"internity/internal/platform/pdfgen"
	"internity/internal/platform/postgres"
	"internity/internal/platform/storage"
)

var errForbidden = httpx.NewError(httpx.ErrForbidden, "You do not have permission to do that")
var errNotFoundAPI = httpx.NewError(httpx.ErrNotFound, "Not found")

type CompanyScopeResolver interface {
	ResolveCompanyScope(ctx context.Context, companyID int64) (schoolID, departmentID int64, err error)
}

// OrgLookup pulls the display names scoring needs for certificate rendering
// — a narrow read-only port into the orgs module.
type OrgLookup interface {
	GetCompanyName(ctx context.Context, companyID int64) (string, error)
	GetSchoolName(ctx context.Context, schoolID int64) (string, error)
}

type StudentInfo struct {
	Name string
	NIS  *string
}

// StudentLookup is a narrow read-only port into the identity module.
type StudentLookup interface {
	GetStudentInfo(ctx context.Context, userID string) (*StudentInfo, error)
}

type Service struct {
	repo      *Repository
	companies CompanyScopeResolver
	orgs      OrgLookup
	students  StudentLookup
	storage   *storage.Client
}

func NewService(repo *Repository, companies CompanyScopeResolver, orgs OrgLookup, students StudentLookup, storageClient *storage.Client) *Service {
	return &Service{repo: repo, companies: companies, orgs: orgs, students: students, storage: storageClient}
}

// --- Scores ---

func (s *Service) ListScores(ctx context.Context, actor *identity.User, userID string, companyID int64, params httpx.ListParams) ([]Score, int64, error) {
	if actor.Role == identity.RoleStudent && actor.ID != userID {
		return nil, 0, errForbidden
	}
	if actor.Role != identity.RoleStudent {
		if err := s.assertCanManageCompany(ctx, actor, companyID); err != nil {
			return nil, 0, err
		}
	}
	return s.repo.ListScoresPaged(ctx, userID, companyID, params)
}

type CreateScoreInput struct {
	UserID    string
	CompanyID int64
	Name      string
	Score     int
	Type      ScoreType
}

func (s *Service) CreateScore(ctx context.Context, actor *identity.User, in CreateScoreInput) (*Score, error) {
	if err := s.assertCanManageCompany(ctx, actor, in.CompanyID); err != nil {
		return nil, err
	}
	if in.Score < 0 || in.Score > 100 {
		return nil, httpx.NewError(httpx.ErrValidation, "Score must be between 0 and 100",
			httpx.ErrorDetail{Field: "score", Issue: "must be between 0 and 100"})
	}
	row := &Score{UserID: in.UserID, CompanyID: in.CompanyID, Name: in.Name, Score: in.Score, Type: in.Type}
	if err := s.repo.CreateScore(ctx, row); err != nil {
		return nil, translateWriteErr(err)
	}
	return row, nil
}

func (s *Service) UpdateScore(ctx context.Context, actor *identity.User, id int64, patch ScorePatch) (*Score, error) {
	row, err := s.repo.GetScore(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if err := s.assertCanManageCompany(ctx, actor, row.CompanyID); err != nil {
		return nil, err
	}
	if patch.Score != nil && (*patch.Score < 0 || *patch.Score > 100) {
		return nil, httpx.NewError(httpx.ErrValidation, "Score must be between 0 and 100",
			httpx.ErrorDetail{Field: "score", Issue: "must be between 0 and 100"})
	}
	patch.applyTo(row)
	if err := s.repo.UpdateScore(ctx, row); err != nil {
		return nil, translateWriteErr(err)
	}
	return row, nil
}

func (s *Service) DeleteScore(ctx context.Context, actor *identity.User, id int64) error {
	row, err := s.repo.GetScore(ctx, id)
	if err != nil {
		return translateGetErr(err)
	}
	if err := s.assertCanManageCompany(ctx, actor, row.CompanyID); err != nil {
		return err
	}
	return translateWriteErr(s.repo.DeleteScore(ctx, id))
}

// --- Score predicates ---

func (s *Service) ListScorePredicates(ctx context.Context, actor *identity.User, schoolID int64, params httpx.ListParams) ([]ScorePredicate, int64, error) {
	if err := s.assertCanViewSchool(actor, schoolID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListScorePredicatesPaged(ctx, schoolID, params)
}

func (s *Service) CreateScorePredicate(ctx context.Context, actor *identity.User, row *ScorePredicate) error {
	if err := s.assertCanManageSchool(actor, row.SchoolID); err != nil {
		return err
	}
	if row.Min > row.Max {
		return httpx.NewError(httpx.ErrValidation, "min must be less than or equal to max",
			httpx.ErrorDetail{Field: "min", Issue: "must be <= max"})
	}
	return translateWriteErr(s.repo.CreateScorePredicate(ctx, row))
}

func (s *Service) UpdateScorePredicate(ctx context.Context, actor *identity.User, id int64, patch ScorePredicatePatch) (*ScorePredicate, error) {
	row, err := s.repo.GetScorePredicate(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if err := s.assertCanManageSchool(actor, row.SchoolID); err != nil {
		return nil, err
	}
	patch.applyTo(row)
	if row.Min > row.Max {
		return nil, httpx.NewError(httpx.ErrValidation, "min must be less than or equal to max",
			httpx.ErrorDetail{Field: "min", Issue: "must be <= max"})
	}
	if err := s.repo.UpdateScorePredicate(ctx, row); err != nil {
		return nil, translateWriteErr(err)
	}
	return row, nil
}

func (s *Service) DeleteScorePredicate(ctx context.Context, actor *identity.User, id int64) error {
	row, err := s.repo.GetScorePredicate(ctx, id)
	if err != nil {
		return translateGetErr(err)
	}
	if err := s.assertCanManageSchool(actor, row.SchoolID); err != nil {
		return err
	}
	return translateWriteErr(s.repo.DeleteScorePredicate(ctx, id))
}

// --- Certificate ---

type CertificateResult struct {
	Certificate *Certificate
	PDF         []byte
}

// GenerateCertificate is idempotent: a certificate_number, once issued, is
// never reissued for the same (user, company) — re-requesting re-renders
// the PDF from the current score snapshot but keeps the same number (plan
// section 2.9).
func (s *Service) GenerateCertificate(ctx context.Context, actor *identity.User, userID string, companyID int64) (*CertificateResult, error) {
	if actor.Role == identity.RoleStudent && actor.ID != userID {
		return nil, errForbidden
	}
	if actor.Role != identity.RoleStudent {
		if err := s.assertCanManageCompany(ctx, actor, companyID); err != nil {
			return nil, err
		}
	}

	student, err := s.students.GetStudentInfo(ctx, userID)
	if err != nil {
		return nil, err
	}
	if student.NIS == nil || *student.NIS == "" {
		return nil, httpx.NewError(httpx.ErrConflict, "Student's NIS must be filled in before a certificate can be issued")
	}

	scores, err := s.repo.ListScores(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	if len(scores) == 0 {
		return nil, httpx.NewError(httpx.ErrConflict, "No scores have been entered yet for this placement")
	}

	schoolID, departmentID, err := s.companies.ResolveCompanyScope(ctx, companyID)
	if err != nil {
		return nil, err
	}
	predicates, err := s.repo.ListScorePredicates(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	avg := Average(scores)
	predicate := ResolvePredicate(avg, predicates)
	if predicate == "" {
		predicate = "-"
	}

	cert, err := s.repo.FindCertificate(ctx, userID, companyID)
	if errors.Is(err, ErrNotFound) {
		seq, seqErr := s.repo.NextCertificateSequence(ctx, departmentID, time.Now().Year())
		if seqErr != nil {
			return nil, seqErr
		}
		cert = &Certificate{
			UserID: userID, DepartmentID: departmentID, CompanyID: companyID,
			CertificateNumber: fmt.Sprintf("CERT-%d-%d-%04d", departmentID, time.Now().Year(), seq),
		}
		if err := s.repo.CreateCertificate(ctx, cert); err != nil {
			return nil, translateWriteErr(err)
		}
	} else if err != nil {
		return nil, err
	}

	companyName, err := s.orgs.GetCompanyName(ctx, companyID)
	if err != nil {
		return nil, err
	}
	schoolName, err := s.orgs.GetSchoolName(ctx, schoolID)
	if err != nil {
		return nil, err
	}

	lines := make([]pdfgen.CertificateScoreLine, 0, len(scores))
	for _, sc := range scores {
		lines = append(lines, pdfgen.CertificateScoreLine{Name: sc.Name, Score: sc.Score, Type: string(sc.Type)})
	}

	pdfBytes, err := pdfgen.GenerateCertificate(pdfgen.CertificateData{
		CertificateNumber: cert.CertificateNumber,
		StudentName:       student.Name,
		NIS:               *student.NIS,
		SchoolName:        schoolName,
		CompanyName:       companyName,
		Predicate:         predicate,
		AverageScore:      avg,
		Scores:            lines,
	})
	if err != nil {
		return nil, err
	}

	uploadResult, err := s.storage.Upload(ctx, storage.UploadInput{
		Bucket: storage.BucketDocuments, KeyPrefix: fmt.Sprintf("certificates/%d", cert.ID),
		OriginalFilename: cert.CertificateNumber + ".pdf", Data: pdfBytes,
		AllowedKinds: []string{"pdf"}, MaxBytes: storage.MaxDocBytes,
	})
	if err == nil {
		_ = s.repo.UpdateCertificateFileKey(ctx, cert.ID, uploadResult.Key)
		cert.FileKey = &uploadResult.Key
	}

	return &CertificateResult{Certificate: cert, PDF: pdfBytes}, nil
}

// --- scope helpers ---

func (s *Service) assertCanManageCompany(ctx context.Context, actor *identity.User, companyID int64) error {
	if actor.Role == identity.RoleAdmin {
		return nil
	}
	if actor.Role == identity.RoleMentor {
		if actor.CompanyID == nil || *actor.CompanyID != companyID {
			return errForbidden
		}
		return nil
	}
	if actor.Role != identity.RoleCoordinator {
		return errForbidden
	}
	schoolID, _, err := s.companies.ResolveCompanyScope(ctx, companyID)
	if err != nil {
		return err
	}
	if actor.SchoolID == nil || *actor.SchoolID != schoolID {
		return errForbidden
	}
	return nil
}

func (s *Service) assertCanViewSchool(actor *identity.User, schoolID int64) error {
	if actor.Role == identity.RoleAdmin {
		return nil
	}
	if actor.SchoolID == nil || *actor.SchoolID != schoolID {
		return errForbidden
	}
	return nil
}

func (s *Service) assertCanManageSchool(actor *identity.User, schoolID int64) error {
	if actor.Role == identity.RoleAdmin {
		return nil
	}
	if actor.Role == identity.RoleCoordinator && actor.SchoolID != nil && *actor.SchoolID == schoolID {
		return nil
	}
	return errForbidden
}

func translateGetErr(err error) error {
	if errors.Is(err, ErrNotFound) {
		return errNotFoundAPI
	}
	return err
}

func translateWriteErr(err error) error {
	if err == nil {
		return nil
	}
	if apiErr := postgres.TranslateError(err); apiErr != nil {
		return apiErr
	}
	return err
}
