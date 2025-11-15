package domain

import "time"

type TeamMember struct {
	UserID   string
	Username string
	IsActive bool
}

type Team struct {
	TeamName string
	Members  []TeamMember
}

type User struct {
	UserID   string
	Username string
	TeamName string
	IsActive bool
}

type PullRequestCreate struct {
	PullRequestID   string
	PullRequestName string
	AuthorID        string
}

type PRStatus string

const (
	PRStatusOpen   PRStatus = "OPEN"
	PRStatusMerged PRStatus = "MERGED"
)

type PullRequest struct {
	PullRequestID     string
	PullRequestName   string
	AuthorID          string
	Status            PRStatus
	AssignedReviewers []string
	CreatedAt         *time.Time
	MergedAt          *time.Time
}

type PullRequestShort struct {
	PullRequestID   string
	PullRequestName string
	AuthorID        string
	Status          PRStatus
}

func (pr *PullRequest) IsMerged() bool {
	return pr.Status == PRStatusMerged
}
