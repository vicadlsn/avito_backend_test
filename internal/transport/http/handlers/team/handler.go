package team

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"

	"avito_backend_task/internal/domain"
	"avito_backend_task/internal/transport/http/response"
)

type TeamService interface {
	CreateTeam(ctx context.Context, team domain.Team) (*domain.Team, error)
	GetTeamByName(ctx context.Context, teamName string) (*domain.Team, error)
}

type TeamHandler struct {
	service   TeamService
	lg        *slog.Logger
	validator *validator.Validate
}

func NewTeamHandler(service TeamService, lg *slog.Logger, validator *validator.Validate) *TeamHandler {
	return &TeamHandler{
		service:   service,
		lg:        lg,
		validator: validator,
	}
}

// POST /team/add
func (h *TeamHandler) AddTeam(w http.ResponseWriter, r *http.Request) {
	op := "TeamHandler.AddTeam"
	log := h.lg.With(slog.String("op", op))

	var dto TeamDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		log.Debug("failed to decode request body", slog.String("error", err.Error()))
		response.RespondError(w, response.ErrInvalidRequest)
		return
	}

	if err := h.validator.Struct(dto); err != nil {
		log.Debug("validation failed", slog.String("error", err.Error()))
		response.RespondError(w, response.ErrInvalidRequest)
		return
	}

	team := dtoToTeam(dto)

	createdTeam, err := h.service.CreateTeam(r.Context(), team)
	if err != nil {
		log.Error("failed to create team", slog.Any("error", err))
		response.RespondError(w, err)
		return
	}

	responseDTO := TeamResponse{
		Team: teamToDTO(*createdTeam),
	}

	response.RespondJSON(w, http.StatusCreated, responseDTO)
}

// GET /team/get?team_name
func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	op := "TeamHandler.GetTeam"
	log := h.lg.With(slog.String("op", op))

	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		log.Debug("team_name parameter is required")
		response.RespondError(w, response.ErrInvalidRequest)
		return
	}

	team, err := h.service.GetTeamByName(r.Context(), teamName)
	if err != nil {
		log.Error("failed to get team", slog.String("team_name", teamName), slog.Any("error", err))
		response.RespondError(w, err)
		return
	}

	responseDTO := teamToDTO(*team)
	response.RespondJSON(w, http.StatusOK, responseDTO)
}
