package pullrequests

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"avito_backend_task/internal/domain"
	"avito_backend_task/internal/repository"
	"avito_backend_task/internal/service/pullrequest/mocks"
	dbmocks "avito_backend_task/pkg/db/mocks"
)

func setupTestService() (*PullRequestService, *mocks.PullRequestRepository, *mocks.UserRepository, *dbmocks.MockTransactionManager) {
	prRepo := new(mocks.PullRequestRepository)
	userRepo := new(mocks.UserRepository)
	txManager := dbmocks.NewMockTransactionManager()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	service := NewPullRequestService(prRepo, userRepo, txManager, logger)
	return service, prRepo, userRepo, txManager
}

func TestPullRequestService_CreatePullRequest(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		prCreate      domain.PullRequestCreate
		setupMocks    func(*mocks.PullRequestRepository, *mocks.UserRepository)
		expectedError error
		validate      func(*testing.T, *domain.PullRequest, error)
	}{
		{
			name: "create PR with reviewers",
			prCreate: domain.PullRequestCreate{
				PullRequestID:   "pr1",
				PullRequestName: "PR1",
				AuthorID:        "author1",
			},
			setupMocks: func(prRepo *mocks.PullRequestRepository, userRepo *mocks.UserRepository) {
				author := &domain.User{
					UserID:   "author1",
					Username: "Author1",
					TeamName: "team1",
					IsActive: true,
				}
				candidates := []domain.User{
					{UserID: "reviewer1", Username: "Reviewer1", TeamName: "team1", IsActive: true},
					{UserID: "reviewer2", Username: "Reviewer2", TeamName: "team1", IsActive: true},
					{UserID: "reviewer3", Username: "Reviewer3", TeamName: "team1", IsActive: true},
				}

				userRepo.On("GetByID", mock.Anything, "author1").Return(author, nil)
				userRepo.On("GetActiveByTeam", mock.Anything, "team1", []string{"author1"}).Return(candidates, nil)

				prRepo.On("Exists", mock.Anything, "pr1").Return(false, nil)
				prRepo.On("CreatePullRequest", mock.Anything, mock.AnythingOfType("domain.PullRequestCreate")).Return(now, nil)
				prRepo.On("AssignReviewer", mock.Anything, "pr1", mock.AnythingOfType("string")).Return(nil).Times(2)

				createdPR := &domain.PullRequest{
					PullRequestID:     "pr1",
					PullRequestName:   "PR1",
					AuthorID:          "author1",
					Status:            domain.PRStatusOpen,
					AssignedReviewers: []string{"reviewer1", "reviewer2"},
					CreatedAt:         &now,
				}
				prRepo.On("GetPullRequestByID", mock.Anything, "pr1").Return(createdPR, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, pr *domain.PullRequest, err error) {
				require.NoError(t, err)
				assert.NotNil(t, pr)
				assert.Equal(t, "pr1", pr.PullRequestID)
				assert.Equal(t, domain.PRStatusOpen, pr.Status)
				assert.Len(t, pr.AssignedReviewers, 2)
			},
		},
		{
			name: "create PR with 1 candidate",
			prCreate: domain.PullRequestCreate{
				PullRequestID:   "pr2",
				PullRequestName: "PR2",
				AuthorID:        "author2",
			},
			setupMocks: func(prRepo *mocks.PullRequestRepository, userRepo *mocks.UserRepository) {
				author := &domain.User{
					UserID:   "author2",
					Username: "Author2",
					TeamName: "team2",
					IsActive: true,
				}
				candidates := []domain.User{
					{UserID: "reviewer1", Username: "Reviewer1", TeamName: "team2", IsActive: true},
				}

				userRepo.On("GetByID", mock.Anything, "author2").Return(author, nil)
				userRepo.On("GetActiveByTeam", mock.Anything, "team2", []string{"author2"}).Return(candidates, nil)

				prRepo.On("Exists", mock.Anything, "pr2").Return(false, nil)
				prRepo.On("CreatePullRequest", mock.Anything, mock.AnythingOfType("domain.PullRequestCreate")).Return(now, nil)
				prRepo.On("AssignReviewer", mock.Anything, "pr2", "reviewer1").Return(nil)

				createdPR := &domain.PullRequest{
					PullRequestID:     "pr2",
					PullRequestName:   "PR2",
					AuthorID:          "author2",
					Status:            domain.PRStatusOpen,
					AssignedReviewers: []string{"reviewer1"},
					CreatedAt:         &now,
				}
				prRepo.On("GetPullRequestByID", mock.Anything, "pr2").Return(createdPR, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, pr *domain.PullRequest, err error) {
				require.NoError(t, err)
				assert.NotNil(t, pr)
				assert.Len(t, pr.AssignedReviewers, 1)
			},
		},
		{
			name: "create PR with no candidates",
			prCreate: domain.PullRequestCreate{
				PullRequestID:   "pr3",
				PullRequestName: "PR3",
				AuthorID:        "author3",
			},
			setupMocks: func(prRepo *mocks.PullRequestRepository, userRepo *mocks.UserRepository) {
				author := &domain.User{
					UserID:   "author3",
					Username: "Author3",
					TeamName: "team3",
					IsActive: true,
				}

				userRepo.On("GetByID", mock.Anything, "author3").Return(author, nil)
				userRepo.On("GetActiveByTeam", mock.Anything, "team3", []string{"author3"}).Return([]domain.User{}, nil)

				prRepo.On("Exists", mock.Anything, "pr3").Return(false, nil)
				prRepo.On("CreatePullRequest", mock.Anything, mock.AnythingOfType("domain.PullRequestCreate")).Return(now, nil)

				createdPR := &domain.PullRequest{
					PullRequestID:     "pr3",
					PullRequestName:   "PR3",
					AuthorID:          "author3",
					Status:            domain.PRStatusOpen,
					AssignedReviewers: []string{},
					CreatedAt:         &now,
				}
				prRepo.On("GetPullRequestByID", mock.Anything, "pr3").Return(createdPR, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, pr *domain.PullRequest, err error) {
				require.NoError(t, err)
				assert.NotNil(t, pr)
				assert.Empty(t, pr.AssignedReviewers)
			},
		},
		{
			name: "PR already exists",
			prCreate: domain.PullRequestCreate{
				PullRequestID:   "existing-pr",
				PullRequestName: "existing pr",
				AuthorID:        "author1",
			},
			setupMocks: func(prRepo *mocks.PullRequestRepository, userRepo *mocks.UserRepository) {
				author := &domain.User{
					UserID:   "author1",
					Username: "Author1",
					TeamName: "team1",
					IsActive: true,
				}
				candidates := []domain.User{
					{UserID: "reviewer1", Username: "Reviewer1", TeamName: "team1", IsActive: true},
				}

				userRepo.On("GetByID", mock.Anything, "author1").Return(author, nil)
				userRepo.On("GetActiveByTeam", mock.Anything, "team1", []string{"author1"}).Return(candidates, nil)

				prRepo.On("Exists", mock.Anything, "existing-pr").Return(true, nil)
			},
			expectedError: domain.ErrPRExists,
			validate: func(t *testing.T, pr *domain.PullRequest, err error) {
				require.Error(t, err)
				assert.Nil(t, pr)
				assert.ErrorIs(t, err, domain.ErrPRExists)
			},
		},
		{
			name: "author not found",
			prCreate: domain.PullRequestCreate{
				PullRequestID:   "pr4",
				PullRequestName: "PR4",
				AuthorID:        "not-found",
			},
			setupMocks: func(prRepo *mocks.PullRequestRepository, userRepo *mocks.UserRepository) {
				userRepo.On("GetByID", mock.Anything, "not-found").Return(nil, repository.ErrNotFound)
			},
			expectedError: domain.ErrUserNotFound,
			validate: func(t *testing.T, pr *domain.PullRequest, err error) {
				require.Error(t, err)
				assert.Nil(t, pr)
				assert.ErrorIs(t, err, domain.ErrUserNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, prRepo, userRepo, _ := setupTestService()
			tt.setupMocks(prRepo, userRepo)

			result, err := service.CreatePullRequest(context.Background(), tt.prCreate)

			tt.validate(t, result, err)
			prRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
		})
	}
}

func TestPullRequestService_MergePullRequest(t *testing.T) {
	now := time.Now()
	mergedAt := time.Now()

	tests := []struct {
		name          string
		prID          string
		setupMocks    func(*mocks.PullRequestRepository)
		expectedError error
		validate      func(*testing.T, *domain.PullRequest, error)
	}{
		{
			name: "merge PR",
			prID: "pr1",
			setupMocks: func(prRepo *mocks.PullRequestRepository) {
				prRepo.On("Exists", mock.Anything, "pr1").Return(true, nil)
				prRepo.On("MergePullRequest", mock.Anything, "pr1").Return(nil)

				mergedPR := &domain.PullRequest{
					PullRequestID:     "pr1",
					PullRequestName:   "PR1",
					AuthorID:          "author1",
					Status:            domain.PRStatusMerged,
					AssignedReviewers: []string{"reviewer1"},
					CreatedAt:         &now,
					MergedAt:          &mergedAt,
				}
				prRepo.On("GetPullRequestByID", mock.Anything, "pr1").Return(mergedPR, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, pr *domain.PullRequest, err error) {
				require.NoError(t, err)
				assert.NotNil(t, pr)
				assert.Equal(t, domain.PRStatusMerged, pr.Status)
				assert.NotNil(t, pr.MergedAt)
			},
		},
		{
			name: "merge PR idempotent",
			prID: "pr2",
			setupMocks: func(prRepo *mocks.PullRequestRepository) {
				prRepo.On("Exists", mock.Anything, "pr2").Return(true, nil)
				prRepo.On("MergePullRequest", mock.Anything, "pr2").Return(nil)

				mergedPR := &domain.PullRequest{
					PullRequestID:     "pr2",
					PullRequestName:   "PR2",
					AuthorID:          "author2",
					Status:            domain.PRStatusMerged,
					AssignedReviewers: []string{"reviewer1"},
					CreatedAt:         &now,
					MergedAt:          &mergedAt,
				}
				prRepo.On("GetPullRequestByID", mock.Anything, "pr2").Return(mergedPR, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, pr *domain.PullRequest, err error) {
				require.NoError(t, err)
				assert.NotNil(t, pr)
				assert.Equal(t, domain.PRStatusMerged, pr.Status)
			},
		},
		{
			name: "PR not found",
			prID: "not-found",
			setupMocks: func(prRepo *mocks.PullRequestRepository) {
				prRepo.On("Exists", mock.Anything, "not-found").Return(false, nil)
			},
			expectedError: domain.ErrPRNotFound,
			validate: func(t *testing.T, pr *domain.PullRequest, err error) {
				require.Error(t, err)
				assert.Nil(t, pr)
				assert.ErrorIs(t, err, domain.ErrPRNotFound)
			},
		},
		{
			name: "repository error on merge",
			prID: "pr3",
			setupMocks: func(prRepo *mocks.PullRequestRepository) {
				prRepo.On("Exists", mock.Anything, "pr3").Return(true, nil)
				prRepo.On("MergePullRequest", mock.Anything, "pr3").Return(errors.New("db error"))
			},
			expectedError: nil,
			validate: func(t *testing.T, pr *domain.PullRequest, err error) {
				require.Error(t, err)
				assert.Nil(t, pr)
				assert.Contains(t, err.Error(), "failed to merge PR")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, prRepo, _, _ := setupTestService()
			tt.setupMocks(prRepo)

			result, err := service.MergePullRequest(context.Background(), tt.prID)

			tt.validate(t, result, err)
			prRepo.AssertExpectations(t)
		})
	}
}

func TestPullRequestService_ReassignReviewer(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		prID          string
		oldUserID     string
		setupMocks    func(*mocks.PullRequestRepository, *mocks.UserRepository)
		expectedError error
		validate      func(*testing.T, *domain.PullRequest, string, error)
	}{
		{
			name:      "reassign reviewer",
			prID:      "pr1",
			oldUserID: "reviewer1",
			setupMocks: func(prRepo *mocks.PullRequestRepository, userRepo *mocks.UserRepository) {
				pr := &domain.PullRequest{
					PullRequestID:     "pr1",
					PullRequestName:   "PR1",
					AuthorID:          "author1",
					Status:            domain.PRStatusOpen,
					AssignedReviewers: []string{"reviewer1", "reviewer2"},
					CreatedAt:         &now,
				}
				prRepo.On("GetPullRequestByID", mock.Anything, "pr1").Return(pr, nil).Once()
				prRepo.On("IsReviewerAssigned", mock.Anything, "pr1", "reviewer1").Return(true, nil)

				oldReviewer := &domain.User{
					UserID:   "reviewer1",
					Username: "Reviewer1",
					TeamName: "team1",
					IsActive: true,
				}
				userRepo.On("GetByID", mock.Anything, "reviewer1").Return(oldReviewer, nil)

				candidates := []domain.User{
					{UserID: "reviewer3", Username: "Reviewer3", TeamName: "team1", IsActive: true},
				}
				userRepo.On("GetActiveByTeam", mock.Anything, "team1", []string{"author1", "reviewer1", "reviewer2"}).Return(candidates, nil)

				prRepo.On("RemoveReviewer", mock.Anything, "pr1", "reviewer1").Return(nil)
				prRepo.On("AssignReviewer", mock.Anything, "pr1", "reviewer3").Return(nil)

				updatedPR := &domain.PullRequest{
					PullRequestID:     "pr1",
					PullRequestName:   "PR1",
					AuthorID:          "author1",
					Status:            domain.PRStatusOpen,
					AssignedReviewers: []string{"reviewer2", "reviewer3"},
					CreatedAt:         &now,
				}
				prRepo.On("GetPullRequestByID", mock.Anything, "pr1").Return(updatedPR, nil).Once()
			},
			expectedError: nil,
			validate: func(t *testing.T, pr *domain.PullRequest, newReviewerID string, err error) {
				require.NoError(t, err)
				assert.NotNil(t, pr)
				assert.Equal(t, "reviewer3", newReviewerID)
				assert.Contains(t, pr.AssignedReviewers, "reviewer3")
				assert.NotContains(t, pr.AssignedReviewers, "reviewer1")
			},
		},
		{
			name:      "PR not found",
			prID:      "not-found",
			oldUserID: "reviewer1",
			setupMocks: func(prRepo *mocks.PullRequestRepository, userRepo *mocks.UserRepository) {
				prRepo.On("GetPullRequestByID", mock.Anything, "not-found").Return(nil, repository.ErrNotFound)
			},
			expectedError: domain.ErrPRNotFound,
			validate: func(t *testing.T, pr *domain.PullRequest, newReviewerID string, err error) {
				require.Error(t, err)
				assert.Nil(t, pr)
				assert.Empty(t, newReviewerID)
				assert.ErrorIs(t, err, domain.ErrPRNotFound)
			},
		},
		{
			name:      "PR merged",
			prID:      "pr2",
			oldUserID: "reviewer1",
			setupMocks: func(prRepo *mocks.PullRequestRepository, userRepo *mocks.UserRepository) {
				mergedAt := time.Now()
				pr := &domain.PullRequest{
					PullRequestID:     "pr2",
					PullRequestName:   "PR2",
					AuthorID:          "author2",
					Status:            domain.PRStatusMerged,
					AssignedReviewers: []string{"reviewer1"},
					CreatedAt:         &now,
					MergedAt:          &mergedAt,
				}
				prRepo.On("GetPullRequestByID", mock.Anything, "pr2").Return(pr, nil)
			},
			expectedError: domain.ErrPRMerged,
			validate: func(t *testing.T, pr *domain.PullRequest, newReviewerID string, err error) {
				require.Error(t, err)
				assert.Nil(t, pr)
				assert.Empty(t, newReviewerID)
				assert.ErrorIs(t, err, domain.ErrPRMerged)
			},
		},
		{
			name:      "reviewer not assigned",
			prID:      "pr3",
			oldUserID: "not-assigned",
			setupMocks: func(prRepo *mocks.PullRequestRepository, userRepo *mocks.UserRepository) {
				pr := &domain.PullRequest{
					PullRequestID:     "pr3",
					PullRequestName:   "PR3",
					AuthorID:          "author3",
					Status:            domain.PRStatusOpen,
					AssignedReviewers: []string{"reviewer1"},
					CreatedAt:         &now,
				}
				prRepo.On("GetPullRequestByID", mock.Anything, "pr3").Return(pr, nil)
				prRepo.On("IsReviewerAssigned", mock.Anything, "pr3", "not-assigned").Return(false, nil)
			},
			expectedError: domain.ErrNotAssigned,
			validate: func(t *testing.T, pr *domain.PullRequest, newReviewerID string, err error) {
				require.Error(t, err)
				assert.Nil(t, pr)
				assert.Empty(t, newReviewerID)
				assert.ErrorIs(t, err, domain.ErrNotAssigned)
			},
		},
		{
			name:      "no candidates",
			prID:      "pr4",
			oldUserID: "reviewer1",
			setupMocks: func(prRepo *mocks.PullRequestRepository, userRepo *mocks.UserRepository) {
				pr := &domain.PullRequest{
					PullRequestID:     "pr4",
					PullRequestName:   "PR4",
					AuthorID:          "author4",
					Status:            domain.PRStatusOpen,
					AssignedReviewers: []string{"reviewer1"},
					CreatedAt:         &now,
				}
				prRepo.On("GetPullRequestByID", mock.Anything, "pr4").Return(pr, nil)
				prRepo.On("IsReviewerAssigned", mock.Anything, "pr4", "reviewer1").Return(true, nil)

				oldReviewer := &domain.User{
					UserID:   "reviewer1",
					Username: "Reviewer1",
					TeamName: "team4",
					IsActive: true,
				}
				userRepo.On("GetByID", mock.Anything, "reviewer1").Return(oldReviewer, nil)

				userRepo.On("GetActiveByTeam", mock.Anything, "team4", []string{"author4", "reviewer1"}).Return([]domain.User{}, nil)
			},
			expectedError: domain.ErrNoCandidate,
			validate: func(t *testing.T, pr *domain.PullRequest, newReviewerID string, err error) {
				require.Error(t, err)
				assert.Nil(t, pr)
				assert.Empty(t, newReviewerID)
				assert.ErrorIs(t, err, domain.ErrNoCandidate)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, prRepo, userRepo, _ := setupTestService()
			tt.setupMocks(prRepo, userRepo)

			result, newReviewerID, err := service.ReassignReviewer(context.Background(), tt.prID, tt.oldUserID)

			tt.validate(t, result, newReviewerID, err)
			prRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
		})
	}
}
