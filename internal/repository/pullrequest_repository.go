package repository

import (
	"context"
	"fmt"
	"time"

	"avito_backend_task/internal/domain"
	"avito_backend_task/pkg/db"
)

type PullRequestRepository struct {
	db *db.DB
}

func NewPullRequestRepository(db *db.DB) *PullRequestRepository {
	return &PullRequestRepository{db: db}
}

func (r *PullRequestRepository) CreatePullRequest(ctx context.Context, pr domain.PullRequestCreate) (time.Time, error) {
	conn := r.db.Conn(ctx)

	var createdAt time.Time
	err := conn.QueryRow(ctx, `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, domain.PRStatusOpen).Scan(&createdAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to insert PR: %w", err)
	}

	return createdAt, nil
}

func (r *PullRequestRepository) Exists(ctx context.Context, prID string) (bool, error) {
	conn := r.db.Conn(ctx)
	var exists bool
	err := conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)", prID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check pr existence: %w", err)
	}
	return exists, nil
}

func (r *PullRequestRepository) AssignReviewer(ctx context.Context, prID, reviewerID string) error {
	conn := r.db.Conn(ctx)

	_, err := conn.Exec(ctx, `
		INSERT INTO pr_reviewers (pull_request_id, user_id)
		VALUES ($1, $2)
	`, prID, reviewerID)
	if err != nil {
		return fmt.Errorf("failed to assign reviewer %s: %w", reviewerID, err)
	}

	return nil
}

func (r *PullRequestRepository) GetPullRequestByID(ctx context.Context, prID string) (*domain.PullRequest, error) {
	conn := r.db.Conn(ctx)

	var pr domain.PullRequest
	var status string
	err := conn.QueryRow(ctx, `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE pull_request_id = $1
	`, prID).Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &status, &pr.CreatedAt, &pr.MergedAt)

	if err != nil {
		return nil, HandleDBError(err)
	}

	pr.Status = domain.PRStatus(status)

	rows, err := conn.Query(ctx, `
		SELECT user_id
		FROM pr_reviewers
		WHERE pull_request_id = $1
	`, prID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reviewers: %w", err)
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, fmt.Errorf("failed to scan reviewer: %w", err)
		}
		reviewers = append(reviewers, reviewerID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	pr.AssignedReviewers = reviewers
	return &pr, nil
}

func (r *PullRequestRepository) MergePullRequest(ctx context.Context, prID string) error {
	conn := r.db.Conn(ctx)
	now := time.Now()

	_, err := conn.Exec(ctx, `
		UPDATE pull_requests
		SET status = $1, merged_at = $2
		WHERE pull_request_id = $3
	`, domain.PRStatusMerged, now, prID)

	if err != nil {
		return fmt.Errorf("failed to update PR status: %w", err)
	}

	return nil
}

func (r *PullRequestRepository) RemoveReviewer(ctx context.Context, prID, reviewerID string) error {
	conn := r.db.Conn(ctx)

	_, err := conn.Exec(ctx, `
		DELETE FROM pr_reviewers
		WHERE pull_request_id = $1 AND user_id = $2
	`, prID, reviewerID)
	if err != nil {
		return fmt.Errorf("failed to delete reviewer: %w", err)
	}

	return nil
}

func (r *PullRequestRepository) GetPullRequestsByReviewer(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	conn := r.db.Conn(ctx)
	rows, err := conn.Query(ctx, `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		INNER JOIN pr_reviewers r ON pr.pull_request_id = r.pull_request_id
		WHERE r.user_id = $1
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query PRs: %w", err)
	}
	defer rows.Close()

	var prs []domain.PullRequestShort
	for rows.Next() {
		var pr domain.PullRequestShort
		var status string
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &status); err != nil {
			return nil, fmt.Errorf("failed to scan PR: %w", err)
		}
		pr.Status = domain.PRStatus(status)
		prs = append(prs, pr)
	}

	return prs, rows.Err()
}

func (r *PullRequestRepository) GetOpenPullRequestsByReviewer(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	conn := r.db.Conn(ctx)
	rows, err := conn.Query(ctx, `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		INNER JOIN pr_reviewers r ON pr.pull_request_id = r.pull_request_id
		WHERE r.user_id = $1 AND pr.status = 'OPEN'
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query open PRs: %w", err)
	}
	defer rows.Close()

	var prs []domain.PullRequestShort
	for rows.Next() {
		var pr domain.PullRequestShort
		var status string
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &status); err != nil {
			return nil, fmt.Errorf("failed to scan PR: %w", err)
		}
		pr.Status = domain.PRStatus(status)
		prs = append(prs, pr)
	}

	return prs, rows.Err()
}

func (r *PullRequestRepository) IsReviewerAssigned(ctx context.Context, prID, userID string) (bool, error) {
	conn := r.db.Conn(ctx)
	var exists bool
	err := conn.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pr_reviewers
			WHERE pull_request_id = $1 AND user_id = $2
		)
	`, prID, userID).Scan(&exists)
	return exists, err
}
