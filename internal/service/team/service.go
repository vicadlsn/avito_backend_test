package teams

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"avito_backend_task/internal/domain"
	"avito_backend_task/internal/repository"
	"avito_backend_task/pkg/db"
)

//go:generate mockery --name=TeamRepository --output=./mocks --case=underscore
type TeamRepository interface {
	Create(ctx context.Context, teamName string) error
	Exists(ctx context.Context, teamName string) (bool, error)
	GetTeamByName(ctx context.Context, teamName string) (*domain.Team, error)
}

//go:generate mockery --name=UserRepository --output=./mocks --case=underscore
type UserRepository interface {
	Upsert(ctx context.Context, user domain.TeamMember, teamName string) error
	GetByID(ctx context.Context, userID string) (*domain.User, error)
	SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error)
}

type TeamService struct {
	teamRepo  TeamRepository
	userRepo  UserRepository
	txManager db.TransactionManagerInterface
	lg        *slog.Logger
}

func NewTeamService(teamRepo TeamRepository, userRepo UserRepository,
	txManager db.TransactionManagerInterface, lg *slog.Logger) *TeamService {
	return &TeamService{
		teamRepo:  teamRepo,
		userRepo:  userRepo,
		txManager: txManager,
		lg:        lg,
	}
}

func (s *TeamService) CreateTeam(ctx context.Context, team domain.Team) (*domain.Team, error) {
	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		exists, err := s.teamRepo.Exists(txCtx, team.TeamName)
		if err != nil {
			return fmt.Errorf("failed to check team existence: %w", err)
		}
		if exists {
			return domain.ErrTeamExists
		}

		if err := s.teamRepo.Create(txCtx, team.TeamName); err != nil {
			return fmt.Errorf("failed to create team: %w", err)
		}

		for _, member := range team.Members {
			if err := s.userRepo.Upsert(txCtx, member, team.TeamName); err != nil {
				return fmt.Errorf("failed to add member %s: %w", member.UserID, err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	s.lg.Info("new team created", slog.String("team_name", team.TeamName), slog.Int("members_count", len(team.Members)))

	return &team, nil
}

func (s *TeamService) GetTeamByName(ctx context.Context, teamName string) (*domain.Team, error) {
	team, err := s.teamRepo.GetTeamByName(ctx, teamName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.ErrTeamNotFound
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	return team, nil
}
