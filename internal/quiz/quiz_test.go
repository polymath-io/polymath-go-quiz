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
