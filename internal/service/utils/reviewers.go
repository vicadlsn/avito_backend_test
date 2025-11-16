package utils

import (
	"fmt"
	"math/rand/v2"

	"avito_backend_task/internal/domain"
)

func SelectRandomReviewers(candidates []domain.User, maxCount int) []domain.User {
	if len(candidates) == 0 {
		return []domain.User{}
	}

	count := maxCount
	if len(candidates) < count {
		count = len(candidates)
	}

	result := make([]domain.User, 0, count)
	indices := rand.Perm(len(candidates))

	for i := 0; i < count; i++ {
		result = append(result, candidates[indices[i]])
	}

	return result
}

func SelectRandomReviewer(candidates []domain.User) (domain.User, error) {
	if len(candidates) == 0 {
		return domain.User{}, fmt.Errorf("slice len in 0")
	}

	index := rand.IntN(len(candidates))
	return candidates[index], nil
}
