package users

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"avito_backend_task/internal/domain"
	"avito_backend_task/internal/repository"
	"avito_backend_task/internal/service/user/mocks"

	dbmocks "avito_backend_task/pkg/db/mocks"
)

func setupTestService() (*UserService, *mocks.UserRepository, *mocks.PullRequestRepository, *dbmocks.MockTransactionManager) {
	userRepo := new(mocks.UserRepository)
	prRepo := new(mocks.PullRequestRepository)
	txManager := dbmocks.NewMockTransactionManager()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	service := NewUserService(userRepo, prRepo, txManager, logger)
	return service, userRepo, prRepo, txManager
}

func TestUserService_SetIsActive(t *testing.T) {
	setupTestService := func() (*UserService, *mocks.UserRepository, *mocks.PullRequestRepository, *dbmocks.MockTransactionManager) {
		userRepo := new(mocks.UserRepository)
		prRepo := new(mocks.PullRequestRepository)
		txManager := dbmocks.NewMockTransactionManager()
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

		service := NewUserService(userRepo, prRepo, txManager, logger)
		return service, userRepo, prRepo, txManager
	}

	tests := []struct {
		name          string
		userID        string
		isActive      bool
		setupMocks    func(*mocks.UserRepository, *mocks.PullRequestRepository)
		expectedError error
		validate      func(*testing.T, *domain.User, error)
	}{
		{
			name:     "activate active user",
			userID:   "user1",
			isActive: true,
			setupMocks: func(userRepo *mocks.UserRepository, prRepo *mocks.PullRequestRepository) {
				user := &domain.User{
					UserID:   "user1",
					Username: "User1",
					TeamName: "team1",
					IsActive: true,
				}
				userRepo.On("SetIsActive", mock.Anything, "user1", true).Return(user, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, user *domain.User, err error) {
				require.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, "user1", user.UserID)
				assert.True(t, user.IsActive)
			},
		},
		{
			name:     "deactivate user without active PRs",
			userID:   "user2",
			isActive: false,
			setupMocks: func(userRepo *mocks.UserRepository, prRepo *mocks.PullRequestRepository) {
				oldUser := &domain.User{
					UserID:   "user2",
					Username: "User2",
					TeamName: "team1",
					IsActive: true,
				}
				userRepo.On("GetByID", mock.Anything, "user2").Return(oldUser, nil)

				prRepo.On("GetOpenPullRequestsByReviewer", mock.Anything, "user2").Return([]domain.PullRequestShort{}, nil)

				deactivatedUser := &domain.User{
					UserID:   "user2",
					Username: "User2",
					TeamName: "team1",
					IsActive: false,
				}
				userRepo.On("SetIsActive", mock.Anything, "user2", false).Return(deactivatedUser, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, user *domain.User, err error) {
				require.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, "user2", user.UserID)
				assert.False(t, user.IsActive)
			},
		},
		{
			name:     "deactivate user with active PRs and reassign",
			userID:   "user3",
			isActive: false,
			setupMocks: func(userRepo *mocks.UserRepository, prRepo *mocks.PullRequestRepository) {
				oldUser := &domain.User{
					UserID:   "user3",
					Username: "User3",
					TeamName: "team1",
					IsActive: true,
				}
				userRepo.On("GetByID", mock.Anything, "user3").Return(oldUser, nil)

				activePRs := []domain.PullRequestShort{
					{PullRequestID: "pr1", PullRequestName: "PR1", AuthorID: "author1", Status: "OPEN"},
				}
				prRepo.On("GetOpenPullRequestsByReviewer", mock.Anything, "user3").Return(activePRs, nil)

				fullPR := &domain.PullRequest{
					PullRequestID:     "pr1",
					PullRequestName:   "PR1",
					AuthorID:          "author1",
					Status:            "OPEN",
					AssignedReviewers: []string{"user3", "reviewer2"},
				}
				prRepo.On("GetPullRequestByID", mock.Anything, "pr1").Return(fullPR, nil)

				candidates := []domain.User{
					{UserID: "candidate1", Username: "Candidate1", TeamName: "team1", IsActive: true},
				}
				userRepo.On("GetActiveByTeam", mock.Anything, "team1", []string{"author1", "user3", "reviewer2"}).Return(candidates, nil)

				prRepo.On("RemoveReviewer", mock.Anything, "pr1", "user3").Return(nil)
				prRepo.On("AssignReviewer", mock.Anything, "pr1", "candidate1").Return(nil)

				deactivatedUser := &domain.User{
					UserID:   "user3",
					Username: "User3",
					TeamName: "team1",
					IsActive: false,
				}
				userRepo.On("SetIsActive", mock.Anything, "user3", false).Return(deactivatedUser, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, user *domain.User, err error) {
				require.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, "user3", user.UserID)
				assert.False(t, user.IsActive)
			},
		},
		{
			name:     "deactivate user already inactive",
			userID:   "user4",
			isActive: false,
			setupMocks: func(userRepo *mocks.UserRepository, prRepo *mocks.PullRequestRepository) {
				inactiveUser := &domain.User{
					UserID:   "user4",
					Username: "User4",
					TeamName: "team1",
					IsActive: false,
				}
				userRepo.On("GetByID", mock.Anything, "user4").Return(inactiveUser, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, user *domain.User, err error) {
				require.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, "user4", user.UserID)
				assert.False(t, user.IsActive)
			},
		},
		{
			name:     "user not found during deactivation",
			userID:   "not-found",
			isActive: false,
			setupMocks: func(userRepo *mocks.UserRepository, prRepo *mocks.PullRequestRepository) {
				userRepo.On("GetByID", mock.Anything, "not-found").Return(nil, repository.ErrNotFound)
			},
			expectedError: domain.ErrUserNotFound,
			validate: func(t *testing.T, user *domain.User, err error) {
				require.Error(t, err)
				assert.Nil(t, user)
				assert.ErrorIs(t, err, domain.ErrUserNotFound)
			},
		},
		{
			name:     "error getting user during deactivation",
			userID:   "user5",
			isActive: false,
			setupMocks: func(userRepo *mocks.UserRepository, prRepo *mocks.PullRequestRepository) {
				userRepo.On("GetByID", mock.Anything, "user5").Return(nil, errors.New("db error"))
			},
			expectedError: nil,
			validate: func(t *testing.T, user *domain.User, err error) {
				require.Error(t, err)
				assert.Nil(t, user)
				assert.Contains(t, err.Error(), "failed to get user")
			},
		},
		{
			name:     "error getting active PRs during deactivation",
			userID:   "user6",
			isActive: false,
			setupMocks: func(userRepo *mocks.UserRepository, prRepo *mocks.PullRequestRepository) {
				oldUser := &domain.User{
					UserID:   "user6",
					Username: "User6",
					TeamName: "team1",
					IsActive: true,
				}
				userRepo.On("GetByID", mock.Anything, "user6").Return(oldUser, nil)
				prRepo.On("GetOpenPullRequestsByReviewer", mock.Anything, "user6").Return(nil, errors.New("db error"))
			},
			expectedError: nil,
			validate: func(t *testing.T, user *domain.User, err error) {
				require.Error(t, err)
				assert.Nil(t, user)
				assert.Contains(t, err.Error(), "failed to get open PRs for reviewer")
			},
		},
		{
			name:     "remove reviewer during deactivation no candidates",
			userID:   "user7",
			isActive: false,
			setupMocks: func(userRepo *mocks.UserRepository, prRepo *mocks.PullRequestRepository) {
				oldUser := &domain.User{
					UserID:   "user7",
					Username: "User7",
					TeamName: "team1",
					IsActive: true,
				}
				userRepo.On("GetByID", mock.Anything, "user7").Return(oldUser, nil)

				activePRs := []domain.PullRequestShort{
					{PullRequestID: "pr2", PullRequestName: "PR2", AuthorID: "author1", Status: "OPEN"},
				}
				prRepo.On("GetOpenPullRequestsByReviewer", mock.Anything, "user7").Return(activePRs, nil)

				fullPR := &domain.PullRequest{
					PullRequestID:     "pr2",
					PullRequestName:   "PR2",
					AuthorID:          "author1",
					Status:            "OPEN",
					AssignedReviewers: []string{"user7", "reviewer2"},
				}
				prRepo.On("GetPullRequestByID", mock.Anything, "pr2").Return(fullPR, nil)

				userRepo.On("GetActiveByTeam", mock.Anything, "team1", []string{"author1", "user7", "reviewer2"}).Return([]domain.User{}, nil)

				prRepo.On("RemoveReviewer", mock.Anything, "pr2", "user7").Return(nil)

				deactivatedUser := &domain.User{
					UserID:   "user7",
					Username: "User7",
					TeamName: "team1",
					IsActive: false,
				}
				userRepo.On("SetIsActive", mock.Anything, "user7", false).Return(deactivatedUser, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, user *domain.User, err error) {
				require.NoError(t, err)
				assert.NotNil(t, user)
				assert.False(t, user.IsActive)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, userRepo, prRepo, _ := setupTestService()
			tt.setupMocks(userRepo, prRepo)

			result, err := service.SetIsActive(context.Background(), tt.userID, tt.isActive)

			tt.validate(t, result, err)
			userRepo.AssertExpectations(t)
			prRepo.AssertExpectations(t)
		})
	}
}
func TestUserService_GetReviewPRsByUserID(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		setupMocks    func(*mocks.PullRequestRepository)
		expectedError error
		validate      func(*testing.T, []domain.PullRequestShort, error)
	}{
		{
			name:   "get PRs for reviewer",
			userID: "reviewer1",
			setupMocks: func(prRepo *mocks.PullRequestRepository) {
				prs := []domain.PullRequestShort{
					{
						PullRequestID:   "pr1",
						PullRequestName: "PR1",
						AuthorID:        "author1",
						Status:          domain.PRStatusOpen,
					},
					{
						PullRequestID:   "pr2",
						PullRequestName: "PR2",
						AuthorID:        "author2",
						Status:          domain.PRStatusMerged,
					},
				}
				prRepo.On("GetPullRequestsByReviewer", mock.Anything, "reviewer1").Return(prs, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, prs []domain.PullRequestShort, err error) {
				require.NoError(t, err)
				assert.Len(t, prs, 2)
				assert.Equal(t, "pr1", prs[0].PullRequestID)
				assert.Equal(t, "pr2", prs[1].PullRequestID)
			},
		},
		{
			name:   "no PRs for reviewer",
			userID: "reviewer2",
			setupMocks: func(prRepo *mocks.PullRequestRepository) {
				prRepo.On("GetPullRequestsByReviewer", mock.Anything, "reviewer2").Return([]domain.PullRequestShort{}, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, prs []domain.PullRequestShort, err error) {
				require.NoError(t, err)
				assert.Empty(t, prs)
			},
		},
		{
			name:   "repository error",
			userID: "reviewer3",
			setupMocks: func(prRepo *mocks.PullRequestRepository) {
				prRepo.On("GetPullRequestsByReviewer", mock.Anything, "reviewer3").Return(nil, errors.New("db error"))
			},
			expectedError: nil,
			validate: func(t *testing.T, prs []domain.PullRequestShort, err error) {
				require.Error(t, err)
				assert.Nil(t, prs)
				assert.Contains(t, err.Error(), "failed to get review PRs")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _, prRepo, _ := setupTestService()
			tt.setupMocks(prRepo)

			result, err := service.GetReviewPRsByUserID(context.Background(), tt.userID)

			tt.validate(t, result, err)
			prRepo.AssertExpectations(t)
		})
	}
}
