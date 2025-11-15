package users

import (
	
	"context"
	"errors"
	"fmt"
	"log/slog"

	"avito_backend_task/internal/domain"
	"avito_backend_task/internal/repository"
)

type UserRepository interface {
	SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error)
}

type PullRequestRepository interface {
	GetPullRequestsByReviewer(ctx context.Context, userID string) ([]domain.PullRequestShort, error)
}

type UserService struct {
	userRepo UserRepository
	prRepo   PullRequestRepository
	lg       *slog.Logger
}

func NewUserService(userRepo UserRepository, prRepo PullRequestRepository, lg *slog.Logger) *UserService {
	return &UserService{
		userRepo: userRepo,
		prRepo:   prRepo,
		lg:       lg,
	}
}

func (s *UserService) SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error) {
	user, err := s.userRepo.SetIsActive(ctx, userID, isActive)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to set user active status: %w", err)
	}

	s.lg.Info("user active status updated", slog.String("user_id", userID), slog.Bool("is_active", isActive))
	return user, nil
}

func (s *UserService) GetReviewPRsByUserID(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	prs, err := s.prRepo.GetPullRequestsByReviewer(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get review PRs: %w", err)
	}

	s.lg.Debug("retrieved review PRs", slog.String("user_id", userID), slog.Int("count", len(prs)))
	return prs, nil
}
