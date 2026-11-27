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

func Test_StateCopiesLastAnswer(t *testing.T) {
	q := New(colorQuestions())
	q.SubmitAnswer("red")

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
