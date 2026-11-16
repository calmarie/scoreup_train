package solvedquestions

import (
	"context"
	"errors"
)

var ErrNegativeXP = errors.New("xp must be non-negative")

type SolvedQuestionRepository interface {
	AwardQuestionXP(ctx context.Context, userID int64, questionID int, xp int) (bool, error)
}

type Service struct {
	repository SolvedQuestionRepository
}

func NewService(repository SolvedQuestionRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) AwardQuestionXP(ctx context.Context, userID int64, questionID int, xp int) (bool, error) {
	if xp < 0 {
		return false, ErrNegativeXP
	}
	return s.repository.AwardQuestionXP(ctx, userID, questionID, xp)
}
