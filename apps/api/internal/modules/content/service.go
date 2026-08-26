package content

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"internity/internal/httpx"
	"internity/internal/modules/identity"
	"internity/internal/platform/postgres"
)

var errForbidden = httpx.NewError(httpx.ErrForbidden, "You do not have permission to do that")
var errNotFoundAPI = httpx.NewError(httpx.ErrNotFound, "Not found")

// DepartmentScopeResolver lets news scope-check a department-scoped post
// against the actor's own school, without importing orgs' repository directly.
type DepartmentScopeResolver interface {
	ResolveDepartmentSchool(ctx context.Context, departmentID int64) (schoolID int64, err error)
}

// AudienceResolver backs the publish-notification fan-out — "everyone with
// a stake in this school/department" — without importing identity's
// repository directly.
type AudienceResolver interface {
	ListUserIDsBySchool(ctx context.Context, schoolID int64) ([]string, error)
	ListUserIDsByDepartment(ctx context.Context, departmentID int64) ([]string, error)
}

type Notifier interface {
	SendMany(ctx context.Context, userIDs []string, notifType, title, body string) error
}

type Service struct {
	repo        *Repository
	departments DepartmentScopeResolver
	audience    AudienceResolver
	notifier    Notifier
}

func NewService(repo *Repository, departments DepartmentScopeResolver, audience AudienceResolver, notifier Notifier) *Service {
	return &Service{repo: repo, departments: departments, audience: audience, notifier: notifier}
}

// --- News ---

// ListNews is the staff-facing listing (includes drafts). Non-admin callers
// are pinned to school-scoped posts for their own school only — a coordinator
// managing school X can't browse another school's drafts by passing a
// different scope_id. (Managing an individual department's drafts is done
// via GetNews/UpdateNews directly on a known post ID, which still
// scope-checks through ResolveDepartmentSchool regardless of this listing's
// simplification.)
func (s *Service) ListNews(ctx context.Context, actor *identity.User, scopeType *NewsScopeType, scopeID *int64, params httpx.ListParams) ([]News, int64, error) {
	filter := NewsFilter{ScopeType: scopeType, ScopeID: scopeID}
	if actor.Role != identity.RoleAdmin {
		if actor.SchoolID == nil {
			return nil, 0, errForbidden
		}
		schoolScope := NewsScopeSchool
		filter = NewsFilter{ScopeType: &schoolScope, ScopeID: actor.SchoolID}
	}
	return s.repo.ListNews(ctx, filter, params)
}

// ListPublicNews is the unauthenticated entrypoint (landing page, logged-out
// visitors) — always published-only, regardless of scope filters.
func (s *Service) ListPublicNews(ctx context.Context, params httpx.ListParams) ([]News, int64, error) {
	return s.repo.ListNews(ctx, NewsFilter{PublishedOnly: true}, params)
}

func (s *Service) GetNewsBySlug(ctx context.Context, slug string) (*News, error) {
	row, err := s.repo.GetNewsBySlug(ctx, slug)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if row.Status != NewsPublished {
		return nil, errNotFoundAPI // don't reveal drafts exist via slug guessing
	}
	return row, nil
}

type CreateNewsInput struct {
	ScopeType NewsScopeType
	ScopeID   int64
	Title     string
	Content   string
	ImageKey  *string
	Publish   bool
}

func (s *Service) CreateNews(ctx context.Context, actor *identity.User, in CreateNewsInput) (*News, error) {
	if err := s.assertCanManageScope(ctx, actor, in.ScopeType, in.ScopeID); err != nil {
		return nil, err
	}

	row := &News{
		AuthorID: actor.ID, ScopeType: in.ScopeType, ScopeID: in.ScopeID,
		Title: in.Title, Slug: slugify(in.Title), Content: in.Content, ImageKey: in.ImageKey,
		Status: NewsDraft,
	}
	if in.Publish {
		now := time.Now()
		row.Status = NewsPublished
		row.PublishedAt = &now
	}
	if err := s.repo.CreateNews(ctx, row); err != nil {
		return nil, translateWriteErr(err)
	}

	if in.Publish {
		s.notifyAudience(row)
	}
	return row, nil
}

type NewsPatch struct {
	Title    *string
	Content  *string
	ImageKey *string
	Publish  *bool
}

func (s *Service) UpdateNews(ctx context.Context, actor *identity.User, id int64, patch NewsPatch) (*News, error) {
	row, err := s.repo.GetNews(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if err := s.assertCanManageScope(ctx, actor, row.ScopeType, row.ScopeID); err != nil {
		return nil, err
	}

	wasPublished := row.Status == NewsPublished
	if patch.Title != nil {
		row.Title = *patch.Title
		row.Slug = slugify(*patch.Title)
	}
	if patch.Content != nil {
		row.Content = *patch.Content
	}
	if patch.ImageKey != nil {
		row.ImageKey = patch.ImageKey
	}
	if patch.Publish != nil && *patch.Publish && !wasPublished {
		now := time.Now()
		row.Status = NewsPublished
		row.PublishedAt = &now
	}

	if err := s.repo.UpdateNews(ctx, row); err != nil {
		return nil, translateWriteErr(err)
	}
	if !wasPublished && row.Status == NewsPublished {
		s.notifyAudience(row)
	}
	return row, nil
}

func (s *Service) DeleteNews(ctx context.Context, actor *identity.User, id int64) error {
	row, err := s.repo.GetNews(ctx, id)
	if err != nil {
		return translateGetErr(err)
	}
	if err := s.assertCanManageScope(ctx, actor, row.ScopeType, row.ScopeID); err != nil {
		return err
	}
	return translateWriteErr(s.repo.DeleteNews(ctx, id))
}

// notifyAudience fans out to everyone in scope — potentially hundreds of
// rows for a school-wide post. It's best-effort (errors are swallowed, see
// below) and has no bearing on whether the publish itself succeeded, so it
// runs detached from the request: on its own goroutine with its own
// bounded-lifetime context, rather than blocking the HTTP response on
// however long the fan-out takes (or being cut short when the request
// context is cancelled right after the response is written).
func (s *Service) notifyAudience(n *News) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		var ids []string
		var err error
		if n.ScopeType == NewsScopeSchool {
			ids, err = s.audience.ListUserIDsBySchool(ctx, n.ScopeID)
		} else {
			ids, err = s.audience.ListUserIDsByDepartment(ctx, n.ScopeID)
		}
		if err != nil || len(ids) == 0 {
			return // best-effort — a notification-fanout failure shouldn't fail the publish
		}
		_ = s.notifier.SendMany(ctx, ids, "news_published", "New announcement: "+n.Title, n.Title)
	}()
}

// --- FAQs (no scoping — one shared list; public read, admin write) ---

func (s *Service) ListFAQs(ctx context.Context) ([]FAQ, error) {
	return s.repo.ListFAQs(ctx)
}

func (s *Service) CreateFAQ(ctx context.Context, actor *identity.User, row *FAQ) error {
	if actor.Role != identity.RoleAdmin && actor.Role != identity.RoleCoordinator {
		return errForbidden
	}
	return translateWriteErr(s.repo.CreateFAQ(ctx, row))
}

func (s *Service) UpdateFAQ(ctx context.Context, actor *identity.User, id int64, question, answer *string, sortOrder *int) (*FAQ, error) {
	if actor.Role != identity.RoleAdmin && actor.Role != identity.RoleCoordinator {
		return nil, errForbidden
	}
	row, err := s.repo.GetFAQ(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if question != nil {
		row.Question = *question
	}
	if answer != nil {
		row.Answer = *answer
	}
	if sortOrder != nil {
		row.SortOrder = *sortOrder
	}
	if err := s.repo.UpdateFAQ(ctx, row); err != nil {
		return nil, translateWriteErr(err)
	}
	return row, nil
}

func (s *Service) DeleteFAQ(ctx context.Context, actor *identity.User, id int64) error {
	if actor.Role != identity.RoleAdmin && actor.Role != identity.RoleCoordinator {
		return errForbidden
	}
	return translateWriteErr(s.repo.DeleteFAQ(ctx, id))
}

// --- helpers ---

func (s *Service) assertCanManageScope(ctx context.Context, actor *identity.User, scopeType NewsScopeType, scopeID int64) error {
	if actor.Role == identity.RoleAdmin {
		return nil
	}
	if actor.Role != identity.RoleCoordinator {
		return errForbidden
	}
	if scopeType == NewsScopeSchool {
		if actor.SchoolID == nil || *actor.SchoolID != scopeID {
			return errForbidden
		}
		return nil
	}
	schoolID, err := s.departments.ResolveDepartmentSchool(ctx, scopeID)
	if err != nil {
		return err
	}
	if actor.SchoolID == nil || *actor.SchoolID != schoolID {
		return errForbidden
	}
	return nil
}

func slugify(title string) string {
	lower := strings.ToLower(title)
	var b strings.Builder
	lastDash := false
	for _, r := range lower {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	// A timestamp suffix keeps slugs unique across retitled/duplicate-titled
	// posts without a DB round-trip to check collision first.
	return fmt.Sprintf("%s-%d", slug, time.Now().UnixNano()%100000)
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
