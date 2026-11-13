package question

import (
	"bytes"
	"context"

	"github.com/calmarie/scoreup_train/internal/users"
)

type QuestistionService interface {
	GetQuestion(ctx context.Context) (Question, error)
	SaveAnswer(ctx context.Context, userAns SubmitAnswerRequest, userId int64) (Answer, error)
}

type UserLookup interface {
	GetUserByTelegramID(
		ctx context.Context,
		telegramID int64,
	) (users.User, error)
}

type Service struct {
	repository QuestionRepository
	user       UserLookup
}

func NewService(repository QuestionRepository, user UserLookup) *Service {
	return &Service{
		repository: repository,
		user:       user,
	}
}

func (s *Service) GetQuestion(ctx context.Context) (Question, error) {
	return s.repository.GetQuestion(ctx)
}

func (s *Service) SaveAnswer(ctx context.Context, userAns SubmitAnswerRequest, tgID int64) (Answer, error) {

	correctAns, err := s.repository.GetCorrectAnswer(ctx, userAns.QuestionID)

	if err != nil {
		return Answer{}, err
	}

	isCorrect := bytes.Equal(correctAns, userAns.Answer)

	user, err := s.user.GetUserByTelegramID(ctx, tgID)

	if err != nil {
		return Answer{}, err
	}

	res := Answer{
		User_ID:     user.Id,
		Question_ID: userAns.QuestionID,
		Answer:      userAns.Answer,
		Is_correct:  isCorrect,
	}

	return s.repository.SaveAnswer(ctx, res)
}
