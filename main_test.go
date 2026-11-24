package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_IsAnswerCorrect(t *testing.T) {
	question := Question{CorrectAnswer: "red"}

	tests := []struct {
		name     string
		answer   string
		expected bool
	}{
		{name: "exact answer", answer: "red", expected: true},
		{name: "different case", answer: "RED", expected: true},
		{name: "incorrect answer", answer: "blue", expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isAnswerCorrect(question, test.answer); got != test.expected {
				t.Errorf("isAnswerCorrect(%q) = %t, want %t", test.answer, got, test.expected)
			}
		})
	}
}

func Test_QuizHandlerDisplaysFirstQuestion(t *testing.T) {
	quiz := Quiz{
		Spec: QuizSpec{
			Questions: []Question{{
				Prompt:  "Pick a color",
				Options: []string{"red", "blue"},
			}},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	newQuizHandler(quiz).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", got)
	}
	for _, want := range []string{
		"Pick a color",
		"<li>red</li>",
		"<li>blue</li>",
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("GET / body does not contain %q:\n%s", want, response.Body.String())
		}
	}
}

func Test_QuizHandlerRejectsQuizWithoutQuestions(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	newQuizHandler(Quiz{}).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Errorf("GET / status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

func Test_QuizHandlerProcessesAnswer(t *testing.T) {
	quiz := Quiz{
		Spec: QuizSpec{
			Questions: []Question{{
				Prompt:        "Pick a color",
				CorrectAnswer: "red",
				Options:       []string{"red", "blue"},
			}},
		},
	}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("answer=red"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	newQuizHandler(quiz).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("POST / status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "Phase: Correct") {
		t.Errorf("POST / body does not show correct-answer feedback:\n%s", response.Body.String())
	}
}
