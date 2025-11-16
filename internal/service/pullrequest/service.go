package pullrequests

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"avito_backend_task/internal/domain"
	"avito_backend_task/internal/repository"
	"avito_backend_task/internal/service/utils"
	"avito_backend_task/pkg/db"
)

//go:generate mockery --name=PullRequestRepository --output=./mocks --case=underscore
type PullRequestRepository interface {
	CreatePullRequest(ctx context.Context, pr domain.PullRequestCreate) (time.Time, error)
	Exists(ctx context.Context, prID string) (bool, error)
	AssignReviewer(ctx context.Context, prID, reviewerID string) error
	GetPullRequestByID(ctx context.Context, prID string) (*domain.PullRequest, error)
	MergePullRequest(ctx context.Context, prID string) error
	RemoveReviewer(ctx context.Context, prID, reviewerID string) error
	IsReviewerAssigned(ctx context.Context, prID, userID string) (bool, error)
}

//go:generate mockery --name=UserRepository --output=./mocks --case=underscore
type UserRepository interface {
	GetByID(ctx context.Context, userID string) (*domain.User, error)
	GetActiveByTeam(ctx context.Context, teamName string, excludeUserIDs []string) ([]domain.User, error)
}

type PullRequestService struct {
	prRepo    PullRequestRepository
	userRepo  UserRepository
	txManager db.TransactionManagerInterface
	lg        *slog.Logger
}

func NewPullRequestService(
	prRepo PullRequestRepository,
	userRepo UserRepository,
	txManager db.TransactionManagerInterface,
	lg *slog.Logger,
) *PullRequestService {
	return &PullRequestService{
		prRepo:    prRepo,
		userRepo:  userRepo,
		txManager: txManager,
		lg:        lg,
	}
}

// автоматически назначаются до двух активных ревьюеров из команды автора, исключая самого автора
// пользователь с isACtive=false не должен назначаться на ревью
// автор PR не может быть ревьюером
func (s *PullRequestService) CreatePullRequest(ctx context.Context, prCreate domain.PullRequestCreate) (*domain.PullRequest, error) {
	op := "PullRequestService.CreatePullRequest"
	log := s.lg.With(
		slog.String("op", op),
		slog.String("pr_id", prCreate.PullRequestID),
		slog.String("author_id", prCreate.AuthorID),
	)

	author, err := s.getPRAuthor(ctx, prCreate.AuthorID)
	if err != nil {
		return nil, err
	}
	log.Debug("found author", slog.String("team_name", author.TeamName))

	var pr *domain.PullRequest
	err = s.txManager.Do(ctx, func(txCtx context.Context) error {
		exists, err := s.prRepo.Exists(txCtx, prCreate.PullRequestID)
		if err != nil {
			return fmt.Errorf("failed to check PR existence: %w", err)
		}
		if exists {
			return domain.ErrPRExists
		}

		candidates, err := s.getReviewCandidates(txCtx, author.TeamName, []string{prCreate.AuthorID})
		if err != nil {
			return err
		}
		log.Debug("found candidates", slog.Int("count", len(candidates)))

		reviewers := utils.SelectRandomReviewers(candidates, 2)
		reviewerIDs := make([]string, len(reviewers))
		for i, r := range reviewers {
			reviewerIDs[i] = r.UserID
		}

		log.Debug("selected reviewers", slog.Any("reviewer_ids", reviewerIDs))

		_, err = s.prRepo.CreatePullRequest(txCtx, prCreate)
		if err != nil {
			return fmt.Errorf("failed to create PR: %w", err)
		}

		for _, reviewerID := range reviewerIDs {
			if err := s.prRepo.AssignReviewer(txCtx, prCreate.PullRequestID, reviewerID); err != nil {
				return fmt.Errorf("failed to assign reviewer %s: %w", reviewerID, err)
			}
		}

		createdPR, err := s.prRepo.GetPullRequestByID(txCtx, prCreate.PullRequestID)
		if err != nil {
			return fmt.Errorf("failed to get created PR: %w", err)
		}
		pr = createdPR

		return nil
	})

	if err != nil {
		log.Error("failed to create PR", slog.Any("error", err))
		return nil, err
	}

	log.Info("new PR created")
	return pr, nil
}

func (s *PullRequestService) MergePullRequest(ctx context.Context, prID string) (*domain.PullRequest, error) {
	op := "PullRequestService.MergePullRequest"
	log := s.lg.With(slog.String("op", op), slog.String("pr_id", prID))

	var pr *domain.PullRequest
	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		exists, err := s.prRepo.Exists(txCtx, prID)
		if err != nil {
			return fmt.Errorf("failed to check PR existence: %w", err)
		}
		if !exists {
			return domain.ErrPRNotFound
		}

		if err := s.prRepo.MergePullRequest(txCtx, prID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return domain.ErrPRNotFound
			}
			return fmt.Errorf("failed to merge PR: %w", err)
		}

		mergedPR, err := s.prRepo.GetPullRequestByID(txCtx, prID)
		if err != nil {
			return fmt.Errorf("failed to get merged PR: %w", err)
		}
		pr = mergedPR

		return nil
	})

	if err != nil {
		return nil, err
	}

	log.Info("PR merged")
	return pr, nil
}

// после merge менять список ревьюеров нельзя
func (s *PullRequestService) ReassignReviewer(ctx context.Context, prID, oldUserID string) (*domain.PullRequest, string, error) {
	op := "PullRequestService.ReassignReviewer"
	log := s.lg.With(
		slog.String("op", op),
		slog.String("pr_id", prID),
		slog.String("old_user_id", oldUserID),
	)

	var updatedPR *domain.PullRequest
	var newReviewerID string

	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		pr, err := s.prRepo.GetPullRequestByID(txCtx, prID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return domain.ErrPRNotFound
			}
			return fmt.Errorf("failed to get PR: %w", err)
		}

		if pr.IsMerged() {
			log.Debug("cannot reassign on merged PR")
			return domain.ErrPRMerged
		}

		isAssigned, err := s.prRepo.IsReviewerAssigned(txCtx, prID, oldUserID)
		if err != nil {
			return fmt.Errorf("failed to check reviewer assignment: %w", err)
		}
		if !isAssigned {
			log.Debug("user not assigned as reviewer")
			return domain.ErrNotAssigned
		}

		oldReviewer, err := s.userRepo.GetByID(txCtx, oldUserID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return domain.ErrUserNotFound
			}
			return fmt.Errorf("failed to get old reviewer: %w", err)
		}

		log.Debug("found old reviewer", slog.String("team_name", oldReviewer.TeamName))

		excludeIDs := []string{pr.AuthorID}
		excludeIDs = append(excludeIDs, pr.AssignedReviewers...)

		candidates, err := s.getReviewCandidates(txCtx, oldReviewer.TeamName, excludeIDs)
		if err != nil {
			return err
		}
		log.Debug("found candidates for reassignment", slog.Int("count", len(candidates)))

		if len(candidates) == 0 {
			log.Debug("no active replacement candidates available")
			return domain.ErrNoCandidate
		}

		newReviewer := utils.SelectRandomReviewers(candidates, 1)[0]
		log.Info("selected new reviewer", slog.String("new_user_id", newReviewer.UserID))

		if err := s.prRepo.RemoveReviewer(txCtx, prID, oldUserID); err != nil {
			return fmt.Errorf("failed to remove reviewer: %w", err)
		}

		if err := s.prRepo.AssignReviewer(txCtx, prID, newReviewer.UserID); err != nil {
			return fmt.Errorf("failed to assign new reviewer: %w", err)
		}

		pr, err = s.prRepo.GetPullRequestByID(txCtx, prID)
		if err != nil {
			return fmt.Errorf("failed to get updated PR: %w", err)
		}
		updatedPR = pr
		newReviewerID = newReviewer.UserID

		return nil
	})

	if err != nil {
		return nil, "", err
	}

	log.Info("reviewer reassigned")
	return updatedPR, newReviewerID, nil
}

func (s *PullRequestService) getPRAuthor(ctx context.Context, authorID string) (*domain.User, error) {
	author, err := s.userRepo.GetByID(ctx, authorID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get author: %w", err)
	}

	return author, nil
}

func (s *PullRequestService) getReviewCandidates(ctx context.Context, teamName string, exclude []string) ([]domain.User, error) {
	candidates, err := s.userRepo.GetActiveByTeam(ctx, teamName, exclude)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	return candidates, nil
}
