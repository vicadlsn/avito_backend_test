package users

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"avito_backend_task/internal/domain"
	"avito_backend_task/internal/repository"
	"avito_backend_task/internal/service/utils"
	"avito_backend_task/pkg/db"
)

//go:generate mockery --name=UserRepository --output=./mocks --case=underscore
type UserRepository interface {
	GetActiveByTeam(ctx context.Context, teamName string, excludeUserIDs []string) ([]domain.User, error)
	GetByID(ctx context.Context, userID string) (*domain.User, error)
	SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error)
}

//go:generate mockery --name=PullRequestRepository --output=./mocks --case=underscore
type PullRequestRepository interface {
	GetPullRequestByID(ctx context.Context, prID string) (*domain.PullRequest, error)
	GetPullRequestsByReviewer(ctx context.Context, userID string) ([]domain.PullRequestShort, error)
	GetOpenPullRequestsByReviewer(ctx context.Context, userID string) ([]domain.PullRequestShort, error)
	RemoveReviewer(ctx context.Context, prID, reviewerID string) error
	AssignReviewer(ctx context.Context, prID, reviewerID string) error
}

type UserService struct {
	userRepo  UserRepository
	prRepo    PullRequestRepository
	txManager db.TransactionManagerInterface
	lg        *slog.Logger
}

func NewUserService(
	userRepo UserRepository,
	prRepo PullRequestRepository,
	txManager db.TransactionManagerInterface,
	lg *slog.Logger,
) *UserService {
	return &UserService{
		userRepo:  userRepo,
		prRepo:    prRepo,
		txManager: txManager,
		lg:        lg,
	}
}

func (s *UserService) SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error) {
	if !isActive {
		return s.deactivateUser(ctx, userID)
	}

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

func (s *UserService) deactivateUser(ctx context.Context, userID string) (*domain.User, error) {
	var user *domain.User

	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		oldUser, err := s.userRepo.GetByID(txCtx, userID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return domain.ErrUserNotFound
			}
			return fmt.Errorf("failed to get user: %w", err)
		}

		if !oldUser.IsActive {
			user = oldUser
			return nil
		}

		openPRs, err := s.prRepo.GetOpenPullRequestsByReviewer(txCtx, userID)
		if err != nil {
			return fmt.Errorf("failed to get open PRs for reviewer: %w", err)
		}

		for _, prShort := range openPRs {
			if err := s.handleReviewerReplacement(txCtx, prShort.PullRequestID, userID, oldUser.TeamName); err != nil {
				return fmt.Errorf("failed to handle PR %s: %w", prShort.PullRequestID, err)
			}
		}

		user, err = s.userRepo.SetIsActive(txCtx, userID, false)
		if err != nil {
			return fmt.Errorf("failed to deactivate user: %w", err)
		}

		s.lg.Info("user deactivated",
			slog.String("user_id", userID),
			slog.Int("prs_processed", len(openPRs)))

		return nil
	})

	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to deactivate user: %w", err)
	}

	return user, nil
}

func (s *UserService) handleReviewerReplacement(
	ctx context.Context,
	prID string,
	oldUserID string,
	teamName string,
) error {
	pr, err := s.prRepo.GetPullRequestByID(ctx, prID)
	if err != nil {
		return fmt.Errorf("failed to get PR %s: %w", prID, err)
	}

	excludeIDs := []string{pr.AuthorID}
	excludeIDs = append(excludeIDs, pr.AssignedReviewers...)

	candidates, err := s.userRepo.GetActiveByTeam(ctx, teamName, excludeIDs)
	if err != nil {
		s.lg.Warn("failed to get replacement candidates, removing reviewer",
			slog.String("pr_id", prID),
			slog.String("user_id", oldUserID),
			slog.Any("error", err))
		return s.removeReviewer(ctx, prID, oldUserID)
	}

	if len(candidates) > 0 {
		newReviewer, err := utils.SelectRandomReviewer(candidates)
		if err != nil {
			s.lg.Warn("failed to select reviewer, removing",
				slog.String("pr_id", prID),
				slog.String("user_id", oldUserID))
			return s.removeReviewer(ctx, prID, oldUserID)
		}

		if err := s.prRepo.RemoveReviewer(ctx, prID, oldUserID); err != nil {
			return fmt.Errorf("failed to remove old reviewer: %w", err)
		}

		if err := s.prRepo.AssignReviewer(ctx, prID, newReviewer.UserID); err != nil {
			return fmt.Errorf("failed to assign new reviewer: %w", err)
		}

		s.lg.Info("reviewer reassigned during deactivation",
			slog.String("pr_id", prID),
			slog.String("old_user_id", oldUserID),
			slog.String("new_user_id", newReviewer.UserID))
		return nil
	}

	s.lg.Info("no replacement candidates found, removing reviewer",
		slog.String("pr_id", prID),
		slog.String("user_id", oldUserID))
	return s.removeReviewer(ctx, prID, oldUserID)
}

func (s *UserService) removeReviewer(ctx context.Context, prID, userID string) error {
	if err := s.prRepo.RemoveReviewer(ctx, prID, userID); err != nil {
		return fmt.Errorf("failed to remove reviewer: %w", err)
	}

	s.lg.Info("removed inactive reviewer from PR",
		slog.String("pr_id", prID),
		slog.String("user_id", userID))

	return nil
}

func (s *UserService) GetReviewPRsByUserID(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	prs, err := s.prRepo.GetPullRequestsByReviewer(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get review PRs: %w", err)
	}

	s.lg.Debug("retrieved review PRs", slog.String("user_id", userID), slog.Int("count", len(prs)))
	return prs, nil
}
