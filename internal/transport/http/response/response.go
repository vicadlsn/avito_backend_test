package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type ErrorCode string

const (
	ErrorCodeTeamExists    ErrorCode = "TEAM_EXISTS"
	ErrorCodePRExists      ErrorCode = "PR_EXISTS"
	ErrorCodePRMerged      ErrorCode = "PR_MERGED"
	ErrorCodeNotAssigned   ErrorCode = "NOT_ASSIGNED"
	ErrorCodeNoCandidate   ErrorCode = "NO_CANDIDATE"
	ErrorCodeNotFound      ErrorCode = "NOT_FOUND"
	ErrorCodeBadRequest    ErrorCode = "BAD_REQUEST"
	ErrorCodeInternalError ErrorCode = "INTERNAL_ERROR"
)

type ErrorDetail struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

var (
	ErrTeamExists     = errors.New("team already exists")
	ErrPRExists       = errors.New("pull request already exists")
	ErrPRMerged       = errors.New("pull request is merged")
	ErrNotAssigned    = errors.New("reviewer not assigned")
	ErrNoCandidate    = errors.New("no candidate available")
	ErrNotFound       = errors.New("not found")
	ErrInvalidRequest = errors.New("invalid request")
)

type ErrorMapping struct {
	Code       ErrorCode
	Message    string
	StatusCode int
}

var errorMappings = map[error]ErrorMapping{
	ErrTeamExists: {
		Code:       ErrorCodeTeamExists,
		Message:    "team_name already exists",
		StatusCode: http.StatusBadRequest,
	},
	ErrPRExists: {
		Code:       ErrorCodePRExists,
		Message:    "PR id already exists",
		StatusCode: http.StatusConflict,
	},
	ErrPRMerged: {
		Code:       ErrorCodePRMerged,
		Message:    "cannot reassign on merged PR",
		StatusCode: http.StatusConflict,
	},
	ErrNotAssigned: {
		Code:       ErrorCodeNotAssigned,
		Message:    "reviewer is not assigned to this PR",
		StatusCode: http.StatusConflict,
	},
	ErrNoCandidate: {
		Code:       ErrorCodeNoCandidate,
		Message:    "no active replacement candidate in team",
		StatusCode: http.StatusConflict,
	},
	ErrNotFound: {
		Code:       ErrorCodeNotFound,
		Message:    "resource not found",
		StatusCode: http.StatusNotFound,
	},
	ErrInvalidRequest: {
		Code:       ErrorCodeBadRequest,
		Message:    "invalid request",
		StatusCode: http.StatusBadRequest,
	},
}

func MapError(err error) ErrorMapping {
	for domainErr, mapping := range errorMappings {
		if errors.Is(err, domainErr) {
			return mapping
		}
	}
	return ErrorMapping{
		Code:       ErrorCodeInternalError,
		Message:    "internal server error",
		StatusCode: http.StatusInternalServerError,
	}
}

func RespondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("failed to encode response", slog.Any("error", err))
		}
	}
}

func RespondError(w http.ResponseWriter, err error) {
	mapping := MapError(err)

	response := ErrorResponse{
		Error: ErrorDetail{
			Code:    mapping.Code,
			Message: mapping.Message,
		},
	}

	RespondJSON(w, mapping.StatusCode, response)
}
