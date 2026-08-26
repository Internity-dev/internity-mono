package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"internity/internal/config"
	"internity/internal/httpx"
	"internity/internal/modules/content"
	"internity/internal/modules/identity"
	"internity/internal/modules/internship"
	"internity/internal/modules/notification"
	"internity/internal/modules/orgs"
	"internity/internal/modules/reporting"
	"internity/internal/modules/review"
	"internity/internal/modules/scoring"
	"internity/internal/modules/vacancy"
	"internity/internal/platform/logger"
	"internity/internal/platform/otel"
	"internity/internal/platform/postgres"
	"internity/internal/platform/queue"
	"internity/internal/platform/redisx"
	"internity/internal/platform/storage"
	"internity/internal/server"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

// queueNotifierAdapter satisfies both vacancy.Notifier and content.Notifier
// (identical Send signature) by enqueueing instead of writing the
// notification row inline in the request path — a notification fan-out to
// many recipients (e.g. a published news post) no longer holds up the HTTP
// response while it writes N rows.
type queueNotifierAdapter struct{ enqueuer *queue.Enqueuer }

func (a queueNotifierAdapter) Send(ctx context.Context, userID, notifType, title, body string) error {
	return a.enqueuer.EnqueueNotification(ctx, queue.NotificationSendPayload{
		UserID: userID, Type: notifType, Title: title, Body: body,
	})
}

// SendMany enqueues one job per recipient rather than a single batch job —
// this is an occasional admin action (e.g. publishing news to a whole
// department), not a hot path, and per-recipient jobs mean one bad user ID
// retries/dead-letters on its own instead of blocking the whole fan-out.
func (a queueNotifierAdapter) SendMany(ctx context.Context, userIDs []string, notifType, title, body string) error {
	for _, userID := range userIDs {
		if err := a.Send(ctx, userID, notifType, title, body); err != nil {
			return err
		}
	}
	return nil
}

// queueMailerAdapter satisfies identity.Mailer by enqueueing — the actual
// send (still a logged no-op until a real provider is configured, see
// cmd/worker) happens on the worker process, off the request path.
type queueMailerAdapter struct{ enqueuer *queue.Enqueuer }

func (a queueMailerAdapter) SendPasswordReset(ctx context.Context, toEmail, rawToken string) error {
	return a.enqueuer.EnqueueEmailPasswordReset(ctx, queue.EmailPasswordResetPayload{
		ToEmail: toEmail, RawToken: rawToken,
	})
}

// companyScopeAdapter bridges orgs.Repository's concrete CompanyScope type to
// the narrow (schoolID, departmentID int64, err error) shape vacancy.Service
// depends on — keeps vacancy from importing orgs' repository directly (see
// plan section 2.1: cross-module calls go through an interface, wired here).
type companyScopeAdapter struct{ repo *orgs.Repository }

func (a companyScopeAdapter) ResolveCompanyScope(ctx context.Context, companyID int64) (int64, int64, error) {
	scope, err := a.repo.ResolveCompanyScope(ctx, companyID)
	if err != nil {
		return 0, 0, err
	}
	return scope.SchoolID, scope.DepartmentID, nil
}

func (a companyScopeAdapter) ResolveDepartmentSchool(ctx context.Context, departmentID int64) (int64, error) {
	dept, err := a.repo.GetDepartment(ctx, departmentID)
	if err != nil {
		return 0, err
	}
	return dept.SchoolID, nil
}

// orgLookupAdapter gives the scoring module the display names it needs for
// certificate rendering, without importing orgs' repository directly.
type orgLookupAdapter struct{ repo *orgs.Repository }

func (a orgLookupAdapter) GetCompanyName(ctx context.Context, companyID int64) (string, error) {
	c, err := a.repo.GetCompany(ctx, companyID)
	if err != nil {
		if errors.Is(err, orgs.ErrNotFound) {
			return "", httpx.NewError(httpx.ErrNotFound, "Company not found")
		}
		return "", err
	}
	return c.Name, nil
}

func (a orgLookupAdapter) GetSchoolName(ctx context.Context, schoolID int64) (string, error) {
	sch, err := a.repo.GetSchool(ctx, schoolID)
	if err != nil {
		if errors.Is(err, orgs.ErrNotFound) {
			return "", httpx.NewError(httpx.ErrNotFound, "School not found")
		}
		return "", err
	}
	return sch.Name, nil
}

// studentLookupAdapter gives the scoring module read-only access to a
// student's name/NIS without importing identity's repository directly.
type studentLookupAdapter struct{ repo identity.Repository }

func (a studentLookupAdapter) GetStudentInfo(ctx context.Context, userID string) (*scoring.StudentInfo, error) {
	u, err := a.repo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			return nil, httpx.NewError(httpx.ErrNotFound, "Student not found")
		}
		return nil, err
	}
	return &scoring.StudentInfo{Name: u.Name, NIS: u.NIS}, nil
}

// reportingStudentLookupAdapter is the same read, shaped for the reporting
// module's (structurally distinct but identical-purpose) StudentInfo type.
type reportingStudentLookupAdapter struct{ repo identity.Repository }

func (a reportingStudentLookupAdapter) GetStudentInfo(ctx context.Context, userID string) (*reporting.StudentInfo, error) {
	u, err := a.repo.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			return nil, httpx.NewError(httpx.ErrNotFound, "Student not found")
		}
		return nil, err
	}
	return &reporting.StudentInfo{Name: u.Name, NIS: u.NIS}, nil
}

// departmentScopeAdapter gives the content module a department's school_id
// without importing orgs' repository directly.
type departmentScopeAdapter struct{ repo *orgs.Repository }

func (a departmentScopeAdapter) ResolveDepartmentSchool(ctx context.Context, departmentID int64) (int64, error) {
	dept, err := a.repo.GetDepartment(ctx, departmentID)
	if err != nil {
		if errors.Is(err, orgs.ErrNotFound) {
			return 0, httpx.NewError(httpx.ErrNotFound, "Department not found")
		}
		return 0, err
	}
	return dept.SchoolID, nil
}

// presenceExportAdapter converts internship.Repository's presence-export
// rows into the reporting module's own row type.
type presenceExportAdapter struct{ repo *internship.Repository }

func (a presenceExportAdapter) ListPresencesForExport(ctx context.Context, userID string, companyID int64) ([]reporting.PresenceExportRow, error) {
	rows, err := a.repo.ListPresencesForExport(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	out := make([]reporting.PresenceExportRow, 0, len(rows))
	for _, r := range rows {
		desc := ""
		if r.Description != nil {
			desc = *r.Description
		}
		out = append(out, reporting.PresenceExportRow{
			Date: r.Date, CheckInAt: r.CheckInAt, CheckOutAt: r.CheckOutAt,
			StatusName: r.StatusName, IsApproved: r.IsApproved, Description: desc,
		})
	}
	return out, nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger.Init(cfg.Env)

	shutdownTracing, err := otel.Init(context.Background(), "internity-api", cfg.OTelExporterEndpoint)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init opentelemetry")
	}

	db, err := postgres.Open(cfg.DatabaseURL, cfg.Env == "development")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}

	rdb, err := redisx.Open(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}

	storageClient, err := storage.Open(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioUseSSL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init storage client")
	}

	redisConnOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse redis url for the job queue")
	}
	enqueuer := queue.NewEnqueuer(redisConnOpt)
	defer enqueuer.Close()

	orgsRepo := orgs.NewRepository(db)
	orgsSvc := orgs.NewService(orgsRepo)
	orgsHandler := orgs.NewHandler(orgsSvc)

	identityRepo := identity.NewRepository(db)
	identitySvc := identity.NewService(identityRepo, queueMailerAdapter{enqueuer: enqueuer}, identity.DefaultConfig(), storageClient, companyScopeAdapter{repo: orgsRepo})
	identityHandler := identity.NewHandler(identitySvc, cfg.CookieSecure, cfg.CookieDomain)

	notificationRepo := notification.NewRepository(db)
	notificationSvc := notification.NewService(notificationRepo)
	notificationHandler := notification.NewHandler(notificationSvc)

	internshipRepo := internship.NewRepository(db)
	internshipSvc := internship.NewService(internshipRepo, companyScopeAdapter{repo: orgsRepo}, storageClient, rdb)
	internshipHandler := internship.NewHandler(internshipSvc)

	vacancyRepo := vacancy.NewRepository(db)
	vacancySvc := vacancy.NewService(vacancyRepo, companyScopeAdapter{repo: orgsRepo}, queueNotifierAdapter{enqueuer: enqueuer}, internshipSvc, rdb)
	vacancyHandler := vacancy.NewHandler(vacancySvc)

	scoringRepo := scoring.NewRepository(db)
	scoringSvc := scoring.NewService(scoringRepo, companyScopeAdapter{repo: orgsRepo},
		orgLookupAdapter{repo: orgsRepo}, studentLookupAdapter{repo: identityRepo}, storageClient)
	scoringHandler := scoring.NewHandler(scoringSvc)

	contentRepo := content.NewRepository(db)
	contentSvc := content.NewService(contentRepo, departmentScopeAdapter{repo: orgsRepo}, identityRepo, queueNotifierAdapter{enqueuer: enqueuer})
	contentHandler := content.NewHandler(contentSvc)

	reviewRepo := review.NewRepository(db)
	reviewSvc := review.NewService(reviewRepo, companyScopeAdapter{repo: orgsRepo})
	reviewHandler := review.NewHandler(reviewSvc)

	reportingSvc := reporting.NewService(identityRepo, presenceExportAdapter{repo: internshipRepo},
		companyScopeAdapter{repo: orgsRepo}, orgLookupAdapter{repo: orgsRepo}, reportingStudentLookupAdapter{repo: identityRepo})
	reportingHandler := reporting.NewHandler(reportingSvc)

	engine := server.New(cfg, server.Dependencies{
		DB: db, Redis: rdb,
		Identity: identityHandler, Orgs: orgsHandler,
		Vacancy: vacancyHandler, Notification: notificationHandler,
		Internship:    internshipHandler,
		Scoring:       scoringHandler,
		Content:       contentHandler,
		Review:        reviewHandler,
		Reporting:     reportingHandler,
		Authenticator: identitySvc,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.Port).Str("env", cfg.Env).Msg("api listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
	}
	if err := shutdownTracing(ctx); err != nil {
		log.Error().Err(err).Msg("opentelemetry shutdown failed")
	}
}
