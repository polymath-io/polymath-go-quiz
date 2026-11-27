package quiz

import "testing"

func Test_NewCopiesQuestions(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(questions []Question)
	}{
		{
			name:   "replacing a question",
			mutate: func(questions []Question) { questions[0] = Question{Prompt: "changed"} },
		},
		{
			name:   "changing a question's prompt",
			mutate: func(questions []Question) { questions[0].Prompt = "changed" },
		},
		{
			name:   "changing an option",
			mutate: func(questions []Question) { questions[0].Options[0] = "changed" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			questions := []Question{{
				Prompt:        "Pick a color",
				CorrectAnswer: "red",
				Options:       []string{"red", "blue"},
			}}
			first := New(questions)
			second := New(questions)

			tt.mutate(questions)

			for name, q := range map[string]*Quiz{"first": first, "second": second} {
				spec, _ := q.Snapshot()
				got := spec.Questions[0]
				if got.Prompt != "Pick a color" || got.Options[0] != "red" || got.Options[1] != "blue" {
					t.Errorf("%s quiz changed after %s: %+v", name, tt.name, got)
				}
			}
		})
	}
}

func Test_NewWithoutQuestions(t *testing.T) {
	if spec, _ := New(nil).Snapshot(); len(spec.Questions) != 0 {
		t.Errorf("New(nil) has %d questions, want 0", len(spec.Questions))
	}
}

func colorQuestions() []Question {
	return []Question{
		{Prompt: "Pick a color", CorrectAnswer: "red", Options: []string{"red", "blue"}},
		{Prompt: "Pick an animal", CorrectAnswer: "cat", Options: []string{"cat", "dog"}},
	}
}

func Test_SubmitAnswerReturnsResultingState(t *testing.T) {
	tests := []struct {
		name             string
		questions        []Question
		prior            []string
		answer           string
		wantResult       SubmitResult
		wantQuestion     int
		wantPhase        string
		wantLastAnswer   *AnswerResult
		wantSubmissionID int
	}{
		{
			name:             "correct answer advances",
			questions:        colorQuestions(),
			answer:           "RED",
			wantResult:       SubmitAccepted,
			wantQuestion:     1,
			wantPhase:        QuizPhasePending,
			wantLastAnswer:   &AnswerResult{Correct: true, Message: "well done"},
			wantSubmissionID: 1,
		},
		{
			name:             "incorrect answer on the last question finishes",
			questions:        colorQuestions(),
			prior:            []string{"red"},
			answer:           "dog",
			wantResult:       SubmitAccepted,
			wantQuestion:     1,
			wantPhase:        QuizPhaseDone,
			wantLastAnswer:   &AnswerResult{Correct: false, Message: "Best luck on next"},
			wantSubmissionID: 2,
		},
		{
			name:             "invalid option leaves the state unchanged",
			questions:        colorQuestions(),
			answer:           "green",
			wantResult:       SubmitInvalidOption,
			wantQuestion:     0,
			wantPhase:        QuizPhasePending,
			wantLastAnswer:   nil,
			wantSubmissionID: 0,
		},
		{
			name:             "answer after the quiz is done is not accepted",
			questions:        colorQuestions()[:1],
			prior:            []string{"blue"},
			answer:           "red",
			wantResult:       SubmitDone,
			wantQuestion:     0,
			wantPhase:        QuizPhaseDone,
			wantLastAnswer:   &AnswerResult{Correct: false, Message: "Best luck on next"},
			wantSubmissionID: 1,
		},
		{
			name:             "quiz without questions has no current question",
			questions:        nil,
			answer:           "red",
			wantResult:       SubmitNoQuestion,
			wantQuestion:     0,
			wantPhase:        QuizPhasePending,
			wantSubmissionID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := New(tt.questions)
			for _, answer := range tt.prior {
				q.SubmitAnswer(answer)
			}

			got := q.SubmitAnswer(tt.answer)

			if got.Result != tt.wantResult {
				t.Errorf("Result = %v, want %v", got.Result, tt.wantResult)
			}
			if got.Status.CurrentQuestion != tt.wantQuestion || got.Status.Phase != tt.wantPhase {
				t.Errorf("state = (question %d, %s), want (question %d, %s)",
					got.Status.CurrentQuestion, got.Status.Phase, tt.wantQuestion, tt.wantPhase)
			}
			if got.Spec.SubmissionId != tt.wantSubmissionID {
				t.Errorf("SubmissionId = %d, want %d", got.Spec.SubmissionId, tt.wantSubmissionID)
			}
			assertLastAnswer(t, got.Status.LastAnswer, tt.wantLastAnswer)

			// The returned state must match what a later read sees.
			spec, status := q.Snapshot()
			if spec.SubmissionId != got.Spec.SubmissionId || status.CurrentQuestion != got.Status.CurrentQuestion ||
				status.Phase != got.Status.Phase {
				t.Errorf("Snapshot() = (%d, %d, %s), outcome = (%d, %d, %s)",
					spec.SubmissionId, status.CurrentQuestion, status.Phase,
					got.Spec.SubmissionId, got.Status.CurrentQuestion, got.Status.Phase)
			}
		})
	}
}

func Test_RestartReturnsResetState(t *testing.T) {
	q := New(colorQuestions())
	q.SubmitAnswer("red")
	q.SubmitAnswer("cat")

	spec, status := q.Restart()

	if status.CurrentQuestion != 0 || status.Phase != QuizPhasePending || status.LastAnswer != nil {
		t.Errorf("Restart() status = %+v, want first question, Pending, no last answer", status)
	}
	if spec.SubmissionId != 0 || spec.UserAnswer != "" {
		t.Errorf("Restart() spec = (%d, %q), want (0, \"\")", spec.SubmissionId, spec.UserAnswer)
	}
}

func Test_StateCopiesLastAnswer(t *testing.T) {
	q := New(colorQuestions())
	outcome := q.SubmitAnswer("red")
	outcome.Status.LastAnswer.Correct = false

	_, status := q.Snapshot()
	status.LastAnswer.Message = "changed"

	_, again := q.Snapshot()
	assertLastAnswer(t, again.LastAnswer, &AnswerResult{Correct: true, Message: "well done"})
}

func Test_FormatLastAnswer(t *testing.T) {
	tests := []struct {
		name string
		last *AnswerResult
		want string
	}{
		{name: "before the first answer", last: nil, want: ""},
		{name: "correct", last: &AnswerResult{Correct: true, Message: "well done"}, want: "Phase: Correct\tMessage: well done"},
		{name: "incorrect", last: &AnswerResult{Message: "Best luck on next"}, want: "Phase: Incorrect\tMessage: Best luck on next"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatLastAnswer(tt.last); got != tt.want {
				t.Errorf("FormatLastAnswer() = %q, want %q", got, tt.want)
			}
		})
	}
}

func assertLastAnswer(t *testing.T, got, want *AnswerResult) {
	t.Helper()

	switch {
	case got == nil && want == nil:
	case got == nil || want == nil:
		t.Errorf("LastAnswer = %+v, want %+v", got, want)
	case *got != *want:
		t.Errorf("LastAnswer = %+v, want %+v", *got, *want)
	}
}
