package web

import (
	"encoding/hex"
	"strings"
	"testing"

	"quiz.com/internal/quiz"
)

func Test_QuizStoreCreateAndGet(t *testing.T) {
	var store QuizStore
	session, err := store.NewSession([]quiz.Question{{
		Prompt:        "Pick a letter",
		CorrectAnswer: "a",
		Options:       []string{"a", "b"},
	}})

	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	assertUUIDv4(t, session.ID)
	if session.CSRFToken == "" {
		t.Fatal("NewSession() returned an empty CSRF token")
	}

	tests := []struct {
		name        string
		id          string
		wantSession *Session
		wantFound   bool
	}{
		{
			name:        "created session",
			id:          session.ID,
			wantSession: session,
			wantFound:   true,
		},
		{
			name:      "unknown session",
			id:        "unknown",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := store.Get(tt.id)
			if ok != tt.wantFound {
				t.Fatalf("Get(%q) found = %t, want %t", tt.id, ok, tt.wantFound)
			}
			if got != tt.wantSession {
				t.Errorf("Get(%q) = %p, want %p", tt.id, got, tt.wantSession)
			}
		})
	}
}

func Test_QuizStoreCreatesIndependentSessions(t *testing.T) {
	tests := []struct {
		name              string
		answer            string
		wantFirstFeedback string
	}{
		{
			name:              "correct answer",
			answer:            "a",
			wantFirstFeedback: "well done",
		},
		{
			name:              "incorrect answer",
			answer:            "b",
			wantFirstFeedback: "Best luck on next",
		},
	}

	questions := []quiz.Question{{
		Prompt:        "Pick a letter",
		CorrectAnswer: "a",
		Options:       []string{"a", "b"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var store QuizStore
			first, err := store.NewSession(questions)
			if err != nil {
				t.Fatalf("NewSession() for first session error = %v", err)
			}

			second, err := store.NewSession(questions)
			if err != nil {
				t.Fatalf("NewSession() for second session error = %v", err)
			}

			if first.ID == second.ID {
				t.Fatal("NewSession() returned the same ID for two sessions")
			}

			if first.CSRFToken == second.CSRFToken {
				t.Fatal("NewSession() returned the same CSRF token for two sessions")
			}

			first.Quiz.SubmitAnswer(tt.answer)
			firstSpec, firstStatus := first.Quiz.Snapshot()
			secondSpec, secondStatus := second.Quiz.Snapshot()

			if firstSpec.SubmissionId != 1 || firstStatus.Feedback != tt.wantFirstFeedback {
				t.Errorf("first session after submission = (%d, %q), want (1, %q)",
					firstSpec.SubmissionId, firstStatus.Feedback, tt.wantFirstFeedback)
			}

			if secondSpec.SubmissionId != 0 || secondSpec.UserAnswer != "" ||
				secondStatus.Phase != quiz.QuizPhasePending {
				t.Errorf("second session changed unexpectedly: submission=%d answer=%q phase=%q",
					secondSpec.SubmissionId, secondSpec.UserAnswer, secondStatus.Phase)
			}
		})
	}
}

func assertUUIDv4(t *testing.T, id string) {
	t.Helper()

	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		t.Fatalf("session ID %q is not in UUID format", id)
	}

	decoded, err := hex.DecodeString(strings.ReplaceAll(id, "-", ""))
	if err != nil {
		t.Fatalf("session ID %q contains non-hex characters: %v", id, err)
	}

	if len(decoded) != 16 {
		t.Fatalf("session ID %q decodes to %d bytes, want 16", id, len(decoded))
	}

	if got := decoded[6] >> 4; got != 4 {
		t.Errorf("session ID version = %d, want 4", got)
	}

	if got := decoded[8] >> 6; got != 2 {
		t.Errorf("session ID variant bits = %b, want 10", got)
	}
}
