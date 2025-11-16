package teams

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
	"avito_backend_task/internal/service/team/mocks"
	dbmocks "avito_backend_task/pkg/db/mocks"
)

func setupTestService() (*TeamService, *mocks.TeamRepository, *mocks.UserRepository, *dbmocks.MockTransactionManager) {
	teamRepo := new(mocks.TeamRepository)
	userRepo := new(mocks.UserRepository)
	txManager := dbmocks.NewMockTransactionManager()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	service := NewTeamService(teamRepo, userRepo, txManager, logger)
	return service, teamRepo, userRepo, txManager
}

func TestTeamService_CreateTeam(t *testing.T) {
	tests := []struct {
		name          string
		team          domain.Team
		setupMocks    func(*mocks.TeamRepository, *mocks.UserRepository)
		expectedError error
		validate      func(*testing.T, *domain.Team, error)
	}{
		{
			name: "create team without members",
			team: domain.Team{
				TeamName: "team1",
				Members:  []domain.TeamMember{},
			},
			setupMocks: func(teamRepo *mocks.TeamRepository, userRepo *mocks.UserRepository) {
				teamRepo.On("Exists", mock.Anything, "team1").Return(false, nil)
				teamRepo.On("Create", mock.Anything, "team1").Return(nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, team *domain.Team, err error) {
				require.NoError(t, err)
				assert.NotNil(t, team)
				assert.Equal(t, "team1", team.TeamName)
				assert.Empty(t, team.Members)
			},
		},
		{
			name: "create team with members",
			team: domain.Team{
				TeamName: "team2",
				Members: []domain.TeamMember{
					{UserID: "user1", Username: "User1", IsActive: true},
					{UserID: "user2", Username: "User2", IsActive: true},
				},
			},
			setupMocks: func(teamRepo *mocks.TeamRepository, userRepo *mocks.UserRepository) {
				teamRepo.On("Exists", mock.Anything, "team2").Return(false, nil)
				teamRepo.On("Create", mock.Anything, "team2").Return(nil)
				userRepo.On("Upsert", mock.Anything, domain.TeamMember{UserID: "user1", Username: "User1", IsActive: true}, "team2").Return(nil)
				userRepo.On("Upsert", mock.Anything, domain.TeamMember{UserID: "user2", Username: "User2", IsActive: true}, "team2").Return(nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, team *domain.Team, err error) {
				require.NoError(t, err)
				assert.NotNil(t, team)
				assert.Equal(t, "team2", team.TeamName)
				assert.Len(t, team.Members, 2)
			},
		},
		{
			name: "team already exists",
			team: domain.Team{
				TeamName: "existing-team",
				Members:  []domain.TeamMember{},
			},
			setupMocks: func(teamRepo *mocks.TeamRepository, userRepo *mocks.UserRepository) {
				teamRepo.On("Exists", mock.Anything, "existing-team").Return(true, nil)
			},
			expectedError: domain.ErrTeamExists,
			validate: func(t *testing.T, team *domain.Team, err error) {
				require.Error(t, err)
				assert.Nil(t, team)
				assert.ErrorIs(t, err, domain.ErrTeamExists)
			},
		},
		{
			name: "repository error on exists check",
			team: domain.Team{
				TeamName: "team3",
				Members:  []domain.TeamMember{},
			},
			setupMocks: func(teamRepo *mocks.TeamRepository, userRepo *mocks.UserRepository) {
				teamRepo.On("Exists", mock.Anything, "team3").Return(false, errors.New("db error"))
			},
			expectedError: nil,
			validate: func(t *testing.T, team *domain.Team, err error) {
				require.Error(t, err)
				assert.Nil(t, team)
				assert.Contains(t, err.Error(), "failed to check team existence")
			},
		},
		{
			name: "repository error on create",
			team: domain.Team{
				TeamName: "team4",
				Members:  []domain.TeamMember{},
			},
			setupMocks: func(teamRepo *mocks.TeamRepository, userRepo *mocks.UserRepository) {
				teamRepo.On("Exists", mock.Anything, "team4").Return(false, nil)
				teamRepo.On("Create", mock.Anything, "team4").Return(errors.New("db error"))
			},
			expectedError: nil,
			validate: func(t *testing.T, team *domain.Team, err error) {
				require.Error(t, err)
				assert.Nil(t, team)
				assert.Contains(t, err.Error(), "failed to create team")
			},
		},
		{
			name: "repository error on member upsert",
			team: domain.Team{
				TeamName: "team5",
				Members: []domain.TeamMember{
					{UserID: "user1", Username: "User1", IsActive: true},
				},
			},
			setupMocks: func(teamRepo *mocks.TeamRepository, userRepo *mocks.UserRepository) {
				teamRepo.On("Exists", mock.Anything, "team5").Return(false, nil)
				teamRepo.On("Create", mock.Anything, "team5").Return(nil)
				userRepo.On("Upsert", mock.Anything, mock.Anything, "team5").Return(errors.New("db error"))
			},
			expectedError: nil,
			validate: func(t *testing.T, team *domain.Team, err error) {
				require.Error(t, err)
				assert.Nil(t, team)
				assert.Contains(t, err.Error(), "failed to add member")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, teamRepo, userRepo, _ := setupTestService()
			tt.setupMocks(teamRepo, userRepo)

			result, err := service.CreateTeam(context.Background(), tt.team)

			tt.validate(t, result, err)
			teamRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
		})
	}
}

func TestTeamService_GetTeamByName(t *testing.T) {
	tests := []struct {
		name          string
		teamName      string
		setupMocks    func(*mocks.TeamRepository)
		expectedError error
		validate      func(*testing.T, *domain.Team, error)
	}{
		{
			name:     "get existing team",
			teamName: "team1",
			setupMocks: func(teamRepo *mocks.TeamRepository) {
				team := &domain.Team{
					TeamName: "team1",
					Members: []domain.TeamMember{
						{UserID: "user1", Username: "User1", IsActive: true},
						{UserID: "user2", Username: "User2", IsActive: false},
					},
				}
				teamRepo.On("GetTeamByName", mock.Anything, "team1").Return(team, nil)
			},
			expectedError: nil,
			validate: func(t *testing.T, team *domain.Team, err error) {
				require.NoError(t, err)
				assert.NotNil(t, team)
				assert.Equal(t, "team1", team.TeamName)
				assert.Len(t, team.Members, 2)
			},
		},
		{
			name:     "team not found",
			teamName: "no-team",
			setupMocks: func(teamRepo *mocks.TeamRepository) {
				teamRepo.On("GetTeamByName", mock.Anything, "no-team").Return(nil, repository.ErrNotFound)
			},
			expectedError: domain.ErrTeamNotFound,
			validate: func(t *testing.T, team *domain.Team, err error) {
				require.Error(t, err)
				assert.Nil(t, team)
				assert.ErrorIs(t, err, domain.ErrTeamNotFound)
			},
		},
		{
			name:     "repository error",
			teamName: "team",
			setupMocks: func(teamRepo *mocks.TeamRepository) {
				teamRepo.On("GetTeamByName", mock.Anything, "team").Return(nil, errors.New("db connection error"))
			},
			expectedError: nil,
			validate: func(t *testing.T, team *domain.Team, err error) {
				require.Error(t, err)
				assert.Nil(t, team)
				assert.Contains(t, err.Error(), "failed to get team")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, teamRepo, _, _ := setupTestService()
			tt.setupMocks(teamRepo)

			result, err := service.GetTeamByName(context.Background(), tt.teamName)

			tt.validate(t, result, err)
			teamRepo.AssertExpectations(t)
		})
	}
}
