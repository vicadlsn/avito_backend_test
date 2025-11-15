package transport

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"

	"avito_backend_task/internal/transport/http/handlers/pullrequest"
	"avito_backend_task/internal/transport/http/handlers/team"
	"avito_backend_task/internal/transport/http/handlers/user"
	"avito_backend_task/internal/transport/http/middleware"
)

type Services struct {
	TeamService        team.TeamService
	UserService        user.UserService
	PullRequestService pullrequest.PullRequestService
}

func NewRouter(services Services, lg *slog.Logger, validator *validator.Validate) http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.LoggingMiddleware(lg))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	teamHandler := team.NewTeamHandler(services.TeamService, lg, validator)
	r.Post("/team/add", teamHandler.AddTeam)
	r.Get("/team/get", teamHandler.GetTeam)

	userHandler := user.NewUserHandler(services.UserService, lg, validator)
	r.Post("/users/setIsActive", userHandler.SetIsActive)
	r.Get("/users/getReview", userHandler.GetReview)

	prHandler := pullrequest.NewPullRequestHandler(services.PullRequestService, lg, validator)
	r.Post("/pullRequest/create", prHandler.CreatePullRequest)
	r.Post("/pullRequest/merge", prHandler.MergePullRequest)
	r.Post("/pullRequest/reassign", prHandler.ReassignReviewer)

	return r
}
