package pullrequest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"

	"avito_backend_task/internal/domain"
	"avito_backend_task/internal/transport/http/response"
)

type PullRequestService interface {
	CreatePullRequest(ctx context.Context, pr domain.PullRequestCreate) (*domain.PullRequest, error)
	MergePullRequest(ctx context.Context, prID string) (*domain.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID string, oldUserID string) (*domain.PullRequest, string, error)
}

type PullRequestHandler struct {
	service   PullRequestService
	lg        *slog.Logger
	validator *validator.Validate
}

func NewPullRequestHandler(service PullRequestService, lg *slog.Logger, validator *validator.Validate) *PullRequestHandler {
	return &PullRequestHandler{
		service:   service,
		lg:        lg,
		validator: validator,
	}
}

// POST /pullRequest/create
func (h *PullRequestHandler) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	op := "PullRequestHandler.CreatePullRequest"
	log := h.lg.With(slog.String("op", op))

	var req CreatePullRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Debug("failed to decode request body", slog.String("error", err.Error()))
		response.RespondError(w, response.ErrInvalidRequest)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		log.Debug("validation failed", slog.String("error", err.Error()))
		response.RespondError(w, response.ErrInvalidRequest)
		return
	}

	prCreate := domain.PullRequestCreate{
		PullRequestID:   req.PullRequestID,
		PullRequestName: req.PullRequestName,
		AuthorID:        req.AuthorID,
	}

	pr, err := h.service.CreatePullRequest(r.Context(), prCreate)
	if err != nil {
		log.Error("failed to create pull request", slog.Any("error", err))
		response.RespondError(w, err)
		return
	}

	responseDTO := PullRequestResponse{
		PR: prToDTO(*pr),
	}

	response.RespondJSON(w, http.StatusCreated, responseDTO)
}

// POST /pullRequest/merge
func (h *PullRequestHandler) MergePullRequest(w http.ResponseWriter, r *http.Request) {
	op := "PullRequestHandler.MergePullRequest"
	log := h.lg.With(slog.String("op", op))

	var req MergePullRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Debug("failed to decode request body", slog.String("error", err.Error()))
		response.RespondError(w, response.ErrInvalidRequest)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		log.Debug("validation failed", slog.String("error", err.Error()))
		response.RespondError(w, response.ErrInvalidRequest)
		return
	}

	pr, err := h.service.MergePullRequest(r.Context(), req.PullRequestID)
	if err != nil {
		log.Error("failed to merge pull request", slog.Any("error", err))
		response.RespondError(w, err)
		return
	}

	responseDTO := PullRequestResponse{
		PR: prToDTO(*pr),
	}

	response.RespondJSON(w, http.StatusOK, responseDTO)
}

// POST /pullRequest/reassign
func (h *PullRequestHandler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	op := "PullRequestHandler.ReassignReviewer"
	log := h.lg.With(slog.String("op", op))

	var req ReassignReviewerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Debug("failed to decode request body", slog.String("error", err.Error()))
		response.RespondError(w, response.ErrInvalidRequest)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		log.Debug("validation failed", slog.String("error", err.Error()))
		response.RespondError(w, response.ErrInvalidRequest)
		return
	}

	pr, newReviewerID, err := h.service.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		log.Error("failed to reassign reviewer", slog.Any("error", err))
		response.RespondError(w, err)
		return
	}

	responseDTO := ReassignResponse{
		PR:         prToDTO(*pr),
		ReplacedBy: newReviewerID,
	}

	response.RespondJSON(w, http.StatusOK, responseDTO)
}
