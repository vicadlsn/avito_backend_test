package pullrequest

import (
	"time"

	"avito_backend_task/internal/domain"
)

type CreatePullRequestRequest struct {
	PullRequestID   string `json:"pull_request_id" validate:"required,max=64"`
	PullRequestName string `json:"pull_request_name" validate:"required,max=64"`
	AuthorID        string `json:"author_id" validate:"required,max=64"`
}

type MergePullRequestRequest struct {
	PullRequestID string `json:"pull_request_id" validate:"required,max=64"`
}

type ReassignReviewerRequest struct {
	PullRequestID string `json:"pull_request_id" validate:"required,max=64"`
	OldUserID     string `json:"old_user_id" validate:"required,max=64"`
}

type PullRequestDTO struct {
	PullRequestID     string     `json:"pull_request_id"`
	PullRequestName   string     `json:"pull_request_name"`
	AuthorID          string     `json:"author_id"`
	Status            string     `json:"status"`
	AssignedReviewers []string   `json:"assigned_reviewers"`
	CreatedAt         *time.Time `json:"createdAt,omitempty"`
	MergedAt          *time.Time `json:"mergedAt,omitempty"`
}

type PullRequestResponse struct {
	PR PullRequestDTO `json:"pr"`
}

type ReassignResponse struct {
	PR         PullRequestDTO `json:"pr"`
	ReplacedBy string         `json:"replaced_by"`
}

func prToDTO(pr domain.PullRequest) PullRequestDTO {
	return PullRequestDTO{
		PullRequestID:     pr.PullRequestID,
		PullRequestName:   pr.PullRequestName,
		AuthorID:          pr.AuthorID,
		Status:            string(pr.Status),
		AssignedReviewers: pr.AssignedReviewers,
		CreatedAt:         pr.CreatedAt,
		MergedAt:          pr.MergedAt,
	}
}
