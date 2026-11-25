package quiz

import (
	"strings"
	"sync"
)

type Question struct {
	Prompt        string
	CorrectAnswer string
	Options       []string
}

type QuizSpec struct {
	Questions    []Question
	UserAnswer   string
	SubmissionId int
}

type QuizStatus struct {
	CurrentQuestion int
	Phase           string
	Feedback        string
	LastResult      string
}

type Quiz struct {
	mu     sync.Mutex
	Spec   QuizSpec
	Status QuizStatus
}

const (
	QuizPhasePending   = "Pending"
	QuizPhaseCorrect   = "Correct"
	QuizPhaseIncorrect = "Incorrect"
	QuizPhaseDone      = "Done"
)

func NewQuestion(prompt string, correctAnswer string, options []string) Question {
	return Question{
		Prompt:        prompt,
		CorrectAnswer: correctAnswer,
		Options:       options,
	}
}

func IsAnswerCorrect(question Question, userAnswer string) bool {
	return strings.EqualFold(userAnswer, question.CorrectAnswer)
}

func IsAnswerOption(options []string, answer string) bool {
	for _, option := range options {
		if strings.EqualFold(option, answer) {
			return true
		}
	}
	return false
}

func ShowStatus(status QuizStatus) string {
	if status.Phase == QuizPhasePending || status.Phase == QuizPhaseDone {
		return "Phase: " + status.Phase
	}
	return "Phase: " + status.Phase + "\tMessage: " + status.Feedback
}

func New(questions []Question) *Quiz {
	return &Quiz{
		Spec: QuizSpec{
			Questions:    questions,
			UserAnswer:   "",
			SubmissionId: 0,
		},
		Status: QuizStatus{
			CurrentQuestion: 0,
			Phase:           QuizPhasePending,
			Feedback:        "",
		},
	}
}

func (q *Quiz) Snapshot() (QuizSpec, QuizStatus) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.Spec, q.Status
}

type SubmitResult int

const (
	SubmitAccepted SubmitResult = iota
	SubmitDone
	SubmitNoQuestion
	SubmitInvalidOption
)

// SubmitOutcome describes a submission. Question and LastResult are the state
// the answer was checked against, taken under the same lock.
type SubmitOutcome struct {
	Result     SubmitResult
	Question   Question
	LastResult string
}

func (q *Quiz) SubmitAnswer(answer string) SubmitOutcome {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.Status.Phase == QuizPhaseDone {
		return SubmitOutcome{Result: SubmitDone}
	}
	if q.Status.CurrentQuestion < 0 || q.Status.CurrentQuestion >= len(q.Spec.Questions) {
		return SubmitOutcome{Result: SubmitNoQuestion}
	}

	question := q.Spec.Questions[q.Status.CurrentQuestion]
	if !IsAnswerOption(question.Options, answer) {
		return SubmitOutcome{Result: SubmitInvalidOption, Question: question, LastResult: q.Status.LastResult}
	}

	q.Spec.UserAnswer = answer
	q.Spec.SubmissionId++

	if IsAnswerCorrect(question, answer) {
		q.Status.Phase = QuizPhaseCorrect
		q.Status.Feedback = "well done"
	} else {
		q.Status.Phase = QuizPhaseIncorrect
		q.Status.Feedback = "Best luck on next"
	}
	q.Status.LastResult = ShowStatus(q.Status)

	if q.Status.CurrentQuestion+1 < len(q.Spec.Questions) {
		q.Status.CurrentQuestion++
		q.Status.Phase = QuizPhasePending
	} else {
		q.Status.Phase = QuizPhaseDone
	}
	return SubmitOutcome{Result: SubmitAccepted, Question: question}
}

func (q *Quiz) Restart() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.Spec.UserAnswer = ""
	q.Spec.SubmissionId = 0
	q.Status = QuizStatus{Phase: QuizPhasePending}
}
