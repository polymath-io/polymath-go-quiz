package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const testCSRFToken = "test-csrf-token"

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

	newQuizHandler(&quiz, testCSRFToken).ServeHTTP(response, request)

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
		`name="csrf_token" value="` + testCSRFToken + `"`,
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("GET / body does not contain %q:\n%s", want, response.Body.String())
		}
	}
}

func Test_QuizHandlerRejectsQuizWithoutQuestions(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	newQuizHandler(&Quiz{}, testCSRFToken).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Errorf("GET / status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

func postAnswer(handler http.Handler, answer string) *httptest.ResponseRecorder {
	return postForm(handler, "/", url.Values{
		"answer":     {answer},
		"csrf_token": {testCSRFToken},
	})
}

func postForm(handler http.Handler, target string, values url.Values) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func getPage(handler http.Handler) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	return response
}

func Test_QuizHandlerRedirectsAndAdvancesAfterAnswer(t *testing.T) {
	quiz := Quiz{
		Spec: QuizSpec{
			Questions: []Question{
				{Prompt: "Pick a color", CorrectAnswer: "red", Options: []string{"red", "blue"}},
				{Prompt: "Pick an animal", Options: []string{"cat", "dog"}},
			},
		},
	}
	handler := newQuizHandler(&quiz, testCSRFToken)

	response := postAnswer(handler, "RED")
	if response.Code != http.StatusSeeOther {
		t.Fatalf("POST / status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	if got := response.Header().Get("Location"); got != "/" {
		t.Errorf("Location = %q, want /", got)
	}

	// Refreshing (repeating the GET) must not advance the quiz again.
	for i := 0; i < 2; i++ {
		body := getPage(handler).Body.String()
		for _, want := range []string{"Pick an animal", "Phase: Correct", "well done"} {
			if !strings.Contains(body, want) {
				t.Errorf("GET / body does not contain %q:\n%s", want, body)
			}
		}
	}
}

func Test_QuizHandlerShowsCompletionAfterLastAnswer(t *testing.T) {
	quiz := Quiz{
		Spec: QuizSpec{
			Questions: []Question{{Prompt: "Pick a color", CorrectAnswer: "red", Options: []string{"red", "blue"}}},
		},
	}
	handler := newQuizHandler(&quiz, testCSRFToken)

	if response := postAnswer(handler, "blue"); response.Code != http.StatusSeeOther {
		t.Fatalf("POST / status = %d, want %d", response.Code, http.StatusSeeOther)
	}

	body := getPage(handler).Body.String()
	for _, want := range []string{"Quiz complete", "Phase: Incorrect", "Best luck on next", "Phase: Done"} {
		if !strings.Contains(body, want) {
			t.Errorf("GET / body does not contain %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, `<form method="POST" action="/">`) {
		t.Errorf("completed quiz page should not show the answer form (restart form is fine):\n%s", body)
	}

	// Posting after completion is ignored.
	if response := postAnswer(handler, "red"); response.Code != http.StatusSeeOther {
		t.Errorf("POST / after completion status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	if !strings.Contains(getPage(handler).Body.String(), "Phase: Incorrect") {
		t.Errorf("answer after completion should not change the result")
	}
}

func Test_QuizHandlerRestartResetsQuiz(t *testing.T) {
	quiz := Quiz{
		Spec: QuizSpec{
			Questions: []Question{{Prompt: "Pick a color", CorrectAnswer: "red", Options: []string{"red", "blue"}}},
		},
	}
	handler := newQuizHandler(&quiz, testCSRFToken)
	postAnswer(handler, "red")

	if body := getPage(handler).Body.String(); !strings.Contains(body, "Restart quiz") {
		t.Errorf("completion page should offer a restart button:\n%s", body)
	}

	response := postForm(handler, "/restart", url.Values{"csrf_token": {testCSRFToken}})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("POST /restart status = %d, want %d", response.Code, http.StatusSeeOther)
	}

	body := getPage(handler).Body.String()
	if !strings.Contains(body, "Pick a color") || !strings.Contains(body, `<form method="POST" action="/">`) {
		t.Errorf("restart should show the first question again:\n%s", body)
	}
	if strings.Contains(body, "Phase:") || strings.Contains(body, "Restart quiz") {
		t.Errorf("restart should clear previous feedback:\n%s", body)
	}
}

func Test_QuizHandlerInstancesShareQuizState(t *testing.T) {
	quiz := Quiz{
		Spec: QuizSpec{
			Questions: []Question{
				{Prompt: "Pick a color", CorrectAnswer: "red", Options: []string{"red", "blue"}},
				{Prompt: "Pick an animal", Options: []string{"cat", "dog"}},
			},
		},
	}
	firstHandler := newQuizHandler(&quiz, testCSRFToken)
	secondHandler := newQuizHandler(&quiz, testCSRFToken)

	if response := postAnswer(firstHandler, "red"); response.Code != http.StatusSeeOther {
		t.Fatalf("POST / status = %d, want %d", response.Code, http.StatusSeeOther)
	}

	body := getPage(secondHandler).Body.String()
	if !strings.Contains(body, "Pick an animal") {
		t.Errorf("second handler did not observe the shared quiz state:\n%s", body)
	}
	if quiz.Spec.SubmissionId != 1 {
		t.Errorf("SubmissionId = %d, want 1", quiz.Spec.SubmissionId)
	}
}

func Test_QuizHandlerRejectsAnswerOutsideOptions(t *testing.T) {
	quiz := Quiz{
		Spec: QuizSpec{
			Questions: []Question{
				{Prompt: "Pick a color", CorrectAnswer: "red", Options: []string{"red", "blue"}},
				{Prompt: "Pick an animal", Options: []string{"cat", "dog"}},
			},
		},
	}
	handler := newQuizHandler(&quiz, testCSRFToken)

	if response := postAnswer(handler, "green"); response.Code != http.StatusBadRequest {
		t.Fatalf("POST / with an unavailable answer status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	response := postAnswer(handler, "green")
	for _, want := range []string{"Pick a color", "<li>red</li>", "Please choose one of the listed options.", `name="answer"`} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("invalid answer page does not contain %q:\n%s", want, response.Body.String())
		}
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", got)
	}
	if !strings.Contains(getPage(handler).Body.String(), "Pick a color") {
		t.Error("invalid answer should not advance the quiz")
	}
	if quiz.Spec.SubmissionId != 0 || quiz.Spec.UserAnswer != "" {
		t.Errorf("invalid answer changed quiz state: SubmissionId=%d UserAnswer=%q", quiz.Spec.SubmissionId, quiz.Spec.UserAnswer)
	}
}

func Test_QuizHandlerRequiresCSRFTokenForStateChanges(t *testing.T) {
	quiz := Quiz{
		Spec: QuizSpec{
			Questions: []Question{
				{Prompt: "Pick a color", CorrectAnswer: "red", Options: []string{"red", "blue"}},
				{Prompt: "Pick an animal", Options: []string{"cat", "dog"}},
			},
		},
	}
	handler := newQuizHandler(&quiz, testCSRFToken)

	if response := postForm(handler, "/", url.Values{"answer": {"red"}}); response.Code != http.StatusForbidden {
		t.Fatalf("POST / without CSRF token status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if !strings.Contains(getPage(handler).Body.String(), "Pick a color") {
		t.Fatal("POST / without CSRF token should not advance the quiz")
	}

	postAnswer(handler, "red")
	if response := postForm(handler, "/restart", url.Values{"csrf_token": {"invalid"}}); response.Code != http.StatusForbidden {
		t.Fatalf("POST /restart with invalid CSRF token status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if !strings.Contains(getPage(handler).Body.String(), "Pick an animal") {
		t.Error("POST /restart with invalid CSRF token should not reset the quiz")
	}
}
