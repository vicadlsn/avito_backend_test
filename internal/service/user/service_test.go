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
)

func setupTestService() (*UserService, *mocks.UserRepository, *mocks.PullRequestRepository) {
	userRepo := new(mocks.UserRepository)
	prRepo := new(mocks.PullRequestRepository)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	service := NewUserService(userRepo, prRepo, logger)
	return service, userRepo, prRepo
}

func TestUserService_SetIsActive(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		isActive      bool
		setupMocks    func(*mocks.UserRepository)
		expectedError error
		validate      func(*testing.T, *domain.User, error)
	}{
		{
			name:     "set active true",
			userID:   "user1",
			isActive: true,
			setupMocks: func(userRepo *mocks.UserRepository) {
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
			name:     "set active false",
			userID:   "user2",
			isActive: false,
			setupMocks: func(userRepo *mocks.UserRepository) {
				user := &domain.User{
					UserID:   "user2",
					Username: "User2",
					TeamName: "team1",
					IsActive: false,
				}
				userRepo.On("SetIsActive", mock.Anything, "user2", false).Return(user, nil)
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
			name:     "user not found",
			userID:   "non-existent",
			isActive: true,
			setupMocks: func(userRepo *mocks.UserRepository) {
				userRepo.On("SetIsActive", mock.Anything, "non-existent", true).Return(nil, repository.ErrNotFound)
			},
			expectedError: domain.ErrUserNotFound,
			validate: func(t *testing.T, user *domain.User, err error) {
				require.Error(t, err)
				assert.Nil(t, user)
				assert.ErrorIs(t, err, domain.ErrUserNotFound)
			},
		},
		{
			name:     "repository error",
			userID:   "user3",
			isActive: true,
			setupMocks: func(userRepo *mocks.UserRepository) {
				userRepo.On("SetIsActive", mock.Anything, "user3", true).Return(nil, errors.New("db error"))
			},
			expectedError: nil,
			validate: func(t *testing.T, user *domain.User, err error) {
				require.Error(t, err)
				assert.Nil(t, user)
				assert.Contains(t, err.Error(), "failed to set user active status")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, userRepo, _ := setupTestService()
			tt.setupMocks(userRepo)

			result, err := service.SetIsActive(context.Background(), tt.userID, tt.isActive)

			tt.validate(t, result, err)
			userRepo.AssertExpectations(t)
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
			name:   "success - get PRs for reviewer",
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
			name:   "success - no PRs for reviewer",
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
			name:   "error - repository error",
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
			service, _, prRepo := setupTestService()
			tt.setupMocks(prRepo)

			result, err := service.GetReviewPRsByUserID(context.Background(), tt.userID)

			tt.validate(t, result, err)
			prRepo.AssertExpectations(t)
		})
	}
}
