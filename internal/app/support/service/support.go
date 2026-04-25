package service

type SendQuestionInput struct {
	UserID int64
	Theme  string
	Body   string
}

type Question struct {
	Theme  string
	Header string
	Body   string
}

type GetMyQuestionsInput struct {
	UserID int64
}

type GetMyQuestionsResult struct {
	Questions []Question
}

type ChangeStatusInput struct {
	UserID     int64
	Status     string
	QuestionID int64
}
type AnswerOnQuestionInput struct {
	UserID     int64
	QuestionID int64
	Answer     string
}

type GettAllMessagesInput struct {
	UserID     int64
	QuestionID int64
}
type Message struct {
	IsAdmin bool
	Text    string
}
type GettAllMessagesResult struct {
	Messages []Message
}

type GetAllQuestionsByFilterInput struct {
	Theme  string
	Status string
	UserID int64
}

type GetAllQuestionsByFilterResult struct {
	Questions []Question
}
