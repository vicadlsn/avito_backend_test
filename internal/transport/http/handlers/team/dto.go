package team

import "avito_backend_task/internal/domain"

type TeamMemberDTO struct {
	UserID   string `json:"user_id" validate:"required,max=64"`
	Username string `json:"username" validate:"required,max=64"`
	IsActive bool   `json:"is_active"`
}

type TeamDTO struct {
	TeamName string          `json:"team_name" validate:"required,max=64"`
	Members  []TeamMemberDTO `json:"members" validate:"required,min=1,dive"`
}

type TeamResponse struct {
	Team TeamDTO `json:"team"`
}

func dtoToTeam(dto TeamDTO) domain.Team {
	members := make([]domain.TeamMember, len(dto.Members))
	for i, m := range dto.Members {
		members[i] = domain.TeamMember{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		}
	}
	return domain.Team{
		TeamName: dto.TeamName,
		Members:  members,
	}
}

func teamToDTO(team domain.Team) TeamDTO {
	members := make([]TeamMemberDTO, len(team.Members))
	for i, m := range team.Members {
		members[i] = TeamMemberDTO{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		}
	}
	return TeamDTO{
		TeamName: team.TeamName,
		Members:  members,
	}
}
