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

// AnswerResult is the outcome of the most recently submitted answer.
type AnswerResult struct {
	Correct bool
	Message string
}

type QuizStatus struct {
	CurrentQuestion int
	Phase           string
	Feedback        string
	LastAnswer      *AnswerResult // nil until the first answer
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

// FormatLastAnswer renders the last answer for the HTML page, such as
// "Phase: Correct\tMessage: well done". It returns "" before the first answer.
func FormatLastAnswer(last *AnswerResult) string {
	if last == nil {
		return ""
	}
	phase := QuizPhaseIncorrect
	if last.Correct {
		phase = QuizPhaseCorrect
	}
	return ShowStatus(QuizStatus{Phase: phase, Feedback: last.Message})
}

// New returns a quiz that owns a deep copy of questions, so later changes to
// the caller's slices do not affect it or any other quiz built from them.
func New(questions []Question) *Quiz {
	return &Quiz{
		Spec: QuizSpec{
			Questions:    cloneQuestions(questions),
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
	return q.Spec, q.statusCopy()
}

// statusCopy returns q.Status with its own copy of LastAnswer, so callers
// cannot change the quiz through the pointer. It must be called with q.mu held.
func (q *Quiz) statusCopy() QuizStatus {
	status := q.Status
	if status.LastAnswer != nil {
		last := *status.LastAnswer
		status.LastAnswer = &last
	}
	return status
}

type SubmitResult int

const (
	SubmitAccepted SubmitResult = iota
	SubmitDone
	SubmitNoQuestion
	SubmitInvalidOption
)

// SubmitOutcome describes a submission. Spec and Status are the quiz state
// after the call, taken under the same lock; they are unchanged when the
// answer was not accepted.
type SubmitOutcome struct {
	Result SubmitResult
	Spec   QuizSpec
	Status QuizStatus
}

func (q *Quiz) SubmitAnswer(answer string) SubmitOutcome {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.Status.Phase == QuizPhaseDone {
		return q.outcome(SubmitDone)
	}
	if q.Status.CurrentQuestion < 0 || q.Status.CurrentQuestion >= len(q.Spec.Questions) {
		return q.outcome(SubmitNoQuestion)
	}

	question := q.Spec.Questions[q.Status.CurrentQuestion]
	if !IsAnswerOption(question.Options, answer) {
		return q.outcome(SubmitInvalidOption)
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
	q.Status.LastAnswer = &AnswerResult{
		Correct: q.Status.Phase == QuizPhaseCorrect,
		Message: q.Status.Feedback,
	}

	if q.Status.CurrentQuestion+1 < len(q.Spec.Questions) {
		q.Status.CurrentQuestion++
		q.Status.Phase = QuizPhasePending
	} else {
		q.Status.Phase = QuizPhaseDone
	}
	return q.outcome(SubmitAccepted)
}

// outcome must be called with q.mu held.
func (q *Quiz) outcome(result SubmitResult) SubmitOutcome {
	return SubmitOutcome{Result: result, Spec: q.Spec, Status: q.statusCopy()}
}

// Restart resets the quiz to its first question and returns the new state.
func (q *Quiz) Restart() (QuizSpec, QuizStatus) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.Spec.UserAnswer = ""
	q.Spec.SubmissionId = 0
	q.Status = QuizStatus{Phase: QuizPhasePending}

	return q.Spec, q.statusCopy()
}

func cloneQuestions(questions []Question) []Question {
	if questions == nil {
		return nil
	}

	cloned := make([]Question, len(questions))
	for i, question := range questions {
		cloned[i] = question
		cloned[i].Options = append([]string(nil), question.Options...)
	}

	return cloned
}
