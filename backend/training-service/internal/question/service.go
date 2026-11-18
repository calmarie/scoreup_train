package question

import (
	"bytes"
	"context"
	"fmt"
)

type QuestistionService interface {
	GetQuestion(ctx context.Context) (Question, error)
	SaveAnswer(ctx context.Context, userAns SubmitAnswerRequest) (Answer, error)
}

type Service struct {
	repository     QuestionRepository
	publishAttempt func(context.Context, Answer) error
}

func NewService(repository QuestionRepository, publishAttempt func(context.Context, Answer) error) *Service {
	return &Service{repository: repository, publishAttempt: publishAttempt}
}

func (s *Service) GetQuestion(ctx context.Context) (Question, error) {
	return s.repository.GetQuestion(ctx)
}

// SaveAnswer commits the attempt before publishing it. A publish error leaves the saved answer in the database.
func (s *Service) SaveAnswer(ctx context.Context, userAns SubmitAnswerRequest) (Answer, error) {

	correctAns, err := s.repository.GetCorrectAnswer(ctx, userAns.QuestionID)

	if err != nil {
		return Answer{}, err
	}

	// (hardcoded) Byte equality makes JSON formatting and array order part of answer matching.
	isCorrect := bytes.Equal(correctAns, userAns.Answer)

	res := Answer{
		UserID:     userAns.UserID,
		QuestionID: userAns.QuestionID,
		Answer:     userAns.Answer,
		IsCorrect:  isCorrect,
	}

	saved, err := s.repository.SaveAnswer(ctx, res)
	if err != nil {
		return Answer{}, err
	}
	if s.publishAttempt != nil {
		if err := s.publishAttempt(ctx, saved); err != nil {
			return Answer{}, fmt.Errorf("%w (answer %d): %w", ErrPublishAttempt, saved.ID, err)
		}
	}
	return saved, nil
}
