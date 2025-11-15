DROP INDEX IF EXISTS idx_pr_reviewers_reviewer_id;
DROP TABLE IF EXISTS pr_reviewers;

DROP INDEX IF EXISTS idx_pr_author_id;
DROP INDEX IF EXISTS idx_pr_status;
DROP TABLE IF EXISTS pull_requests;

DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_team_name;
DROP TABLE IF EXISTS users;

DROP TABLE IF EXISTS teams;