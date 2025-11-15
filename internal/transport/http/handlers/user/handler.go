package user

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"

	"avito_backend_task/internal/domain"
	"avito_backend_task/internal/transport/http/response"
)

type UserService interface {
	SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error)
	GetReviewPRsByUserID(ctx context.Context, userID string) ([]domain.PullRequestShort, error)
}

type UserHandler struct {
	service   UserService
	lg        *slog.Logger
	validator *validator.Validate
}

func NewUserHandler(service UserService, lg *slog.Logger, validator *validator.Validate) *UserHandler {
	return &UserHandler{
		service:   service,
		lg:        lg,
		validator: validator,
	}
}

// POST /users/setIsActive
func (h *UserHandler) SetIsActive(w http.ResponseWriter, r *http.Request) {
	op := "UserHandler.SetIsActive"
	log := h.lg.With(slog.String("op", op))

	var req SetIsActiveRequest
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

	user, err := h.service.SetIsActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		log.Error("failed to set user active status", slog.Any("error", err))
		response.RespondError(w, err)
		return
	}

	responseDTO := UserResponse{
		User: userToDTO(*user),
	}

	response.RespondJSON(w, http.StatusOK, responseDTO)
}

// GET /users/getReview?user_id
func (h *UserHandler) GetReview(w http.ResponseWriter, r *http.Request) {
	op := "UserHandler.GetReview"
	log := h.lg.With(slog.String("op", op))

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		log.Debug("user_id parameter is required")
		response.RespondError(w, response.ErrInvalidRequest)
		return
	}

	prs, err := h.service.GetReviewPRsByUserID(r.Context(), userID)
	if err != nil {
		log.Error("failed to get review PRs", slog.String("user_id", userID), slog.Any("error", err))
		response.RespondError(w, err)
		return
	}

	prDTOs := make([]PullRequestShortDTO, len(prs))
	for i, pr := range prs {
		prDTOs[i] = prShortToDTO(pr)
	}

	responseDTO := GetReviewResponse{
		UserID:       userID,
		PullRequests: prDTOs,
	}

	response.RespondJSON(w, http.StatusOK, responseDTO)
}
