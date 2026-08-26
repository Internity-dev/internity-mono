package review

import (
	"context"
	"errors"

	"internity/internal/httpx"
	"internity/internal/modules/identity"
	"internity/internal/platform/postgres"
)

var errForbidden = httpx.NewError(httpx.ErrForbidden, "You do not have permission to do that")
var errNotFoundAPI = httpx.NewError(httpx.ErrNotFound, "Not found")

type CompanyScopeResolver interface {
	ResolveCompanyScope(ctx context.Context, companyID int64) (schoolID, departmentID int64, err error)
}

type Service struct {
	repo      *Repository
	companies CompanyScopeResolver
}

func NewService(repo *Repository, companies CompanyScopeResolver) *Service {
	return &Service{repo: repo, companies: companies}
}

// AverageRating is a pure function — used for a company's aggregate rating display.
func AverageRating(reviews []Review) float64 {
	if len(reviews) == 0 {
		return 0
	}
	var sum int
	for _, r := range reviews {
		sum += r.Rating
	}
	return float64(sum) / float64(len(reviews))
}

// --- Monitors ---

func (s *Service) ListMonitors(ctx context.Context, actor *identity.User, studentID *string, companyID *int64, params httpx.ListParams) ([]Monitor, int64, error) {
	switch actor.Role {
	case identity.RoleAdmin:
	case identity.RoleStudent:
		studentID = &actor.ID
	case identity.RoleMentor:
		companyID = actor.CompanyID
	case identity.RoleCoordinator:
		if studentID == nil && companyID == nil {
			return nil, 0, httpx.NewError(httpx.ErrValidation, "Provide a student_id or company_id filter")
		}
	}
	return s.repo.ListMonitors(ctx, studentID, companyID, params)
}

func (s *Service) CreateMonitor(ctx context.Context, actor *identity.User, row *Monitor) error {
	if actor.Role != identity.RoleCoordinator && actor.Role != identity.RoleAdmin {
		return errForbidden
	}
	if actor.Role == identity.RoleCoordinator {
		if err := s.assertCoordinatorOwnsCompany(ctx, actor, row.CompanyID); err != nil {
			return err
		}
	}
	if row.MatchRating < 1 || row.MatchRating > 4 {
		return httpx.NewError(httpx.ErrValidation, "match_rating must be between 1 and 4",
			httpx.ErrorDetail{Field: "match_rating", Issue: "must be between 1 and 4"})
	}
	row.CoordinatorID = actor.ID
	return translateWriteErr(s.repo.CreateMonitor(ctx, row))
}

func (s *Service) DeleteMonitor(ctx context.Context, actor *identity.User, id int64) error {
	row, err := s.repo.GetMonitor(ctx, id)
	if err != nil {
		return translateGetErr(err)
	}
	if actor.Role != identity.RoleAdmin && row.CoordinatorID != actor.ID {
		return errForbidden
	}
	return translateWriteErr(s.repo.DeleteMonitor(ctx, id))
}

// --- Questions ---

func (s *Service) ListQuestions(ctx context.Context, actor *identity.User, schoolID int64, params httpx.ListParams) ([]Question, int64, error) {
	if actor.Role != identity.RoleAdmin && (actor.SchoolID == nil || *actor.SchoolID != schoolID) {
		return nil, 0, errForbidden
	}
	return s.repo.ListQuestions(ctx, schoolID, params)
}

func (s *Service) CreateQuestion(ctx context.Context, actor *identity.User, row *Question) error {
	if err := s.assertCanManageSchool(actor, row.SchoolID); err != nil {
		return err
	}
	return translateWriteErr(s.repo.CreateQuestion(ctx, row))
}

func (s *Service) UpdateQuestion(ctx context.Context, actor *identity.User, id int64, text *string, sortOrder *int) (*Question, error) {
	row, err := s.repo.GetQuestion(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if err := s.assertCanManageSchool(actor, row.SchoolID); err != nil {
		return nil, err
	}
	if text != nil {
		row.Question = *text
	}
	if sortOrder != nil {
		row.SortOrder = *sortOrder
	}
	if err := s.repo.UpdateQuestion(ctx, row); err != nil {
		return nil, translateWriteErr(err)
	}
	return row, nil
}

func (s *Service) DeleteQuestion(ctx context.Context, actor *identity.User, id int64) error {
	row, err := s.repo.GetQuestion(ctx, id)
	if err != nil {
		return translateGetErr(err)
	}
	if err := s.assertCanManageSchool(actor, row.SchoolID); err != nil {
		return err
	}
	return translateWriteErr(s.repo.DeleteQuestion(ctx, id))
}

// --- Reviews ---

type CreateReviewInput struct {
	RevieweeType      RevieweeType
	RevieweeUserID    *string
	RevieweeCompanyID *int64
	QuestionID        *int64
	Title             *string
	Body              *string
	Rating            int
}

func (s *Service) CreateReview(ctx context.Context, actor *identity.User, in CreateReviewInput) (*Review, error) {
	if in.Rating < 1 || in.Rating > 5 {
		return nil, httpx.NewError(httpx.ErrValidation, "rating must be between 1 and 5",
			httpx.ErrorDetail{Field: "rating", Issue: "must be between 1 and 5"})
	}

	switch in.RevieweeType {
	case RevieweeUser:
		// Mentor rates a student they mentor.
		if actor.Role != identity.RoleMentor {
			return nil, errForbidden
		}
		if in.RevieweeUserID == nil {
			return nil, httpx.NewError(httpx.ErrValidation, "reviewee_user_id is required", httpx.ErrorDetail{Field: "reviewee_user_id", Issue: "required"})
		}
	case RevieweeCompany:
		// Student rates the company they interned at.
		if actor.Role != identity.RoleStudent {
			return nil, errForbidden
		}
		if in.RevieweeCompanyID == nil {
			return nil, httpx.NewError(httpx.ErrValidation, "reviewee_company_id is required", httpx.ErrorDetail{Field: "reviewee_company_id", Issue: "required"})
		}
	default:
		return nil, httpx.NewError(httpx.ErrValidation, "reviewee_type must be 'user' or 'company'",
			httpx.ErrorDetail{Field: "reviewee_type", Issue: "must be user or company"})
	}

	row := &Review{
		ReviewerID: actor.ID, QuestionID: in.QuestionID,
		RevieweeUserID: in.RevieweeUserID, RevieweeCompanyID: in.RevieweeCompanyID,
		Title: in.Title, Body: in.Body, Rating: in.Rating,
	}
	if err := s.repo.CreateReview(ctx, row); err != nil {
		return nil, translateWriteErr(err)
	}
	return row, nil
}

func (s *Service) ListReviewsForUser(ctx context.Context, actor *identity.User, userID string) ([]Review, error) {
	if actor.Role == identity.RoleStudent && actor.ID != userID {
		return nil, errForbidden
	}
	return s.repo.ListReviewsForUser(ctx, userID)
}

// ListReviewsForCompany is intentionally open to any authenticated role —
// company reputation is useful context for every actor browsing vacancies.
func (s *Service) ListReviewsForCompany(ctx context.Context, companyID int64) ([]Review, error) {
	return s.repo.ListReviewsForCompany(ctx, companyID)
}

// --- helpers ---

func (s *Service) assertCoordinatorOwnsCompany(ctx context.Context, actor *identity.User, companyID int64) error {
	schoolID, _, err := s.companies.ResolveCompanyScope(ctx, companyID)
	if err != nil {
		return err
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
