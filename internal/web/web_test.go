package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"quiz.com/internal/quiz"
)

func Test_IsAnswerCorrect(t *testing.T) {
	question := quiz.Question{CorrectAnswer: "red"}

	tests := []struct {
		name     string
		answer   string
		expected bool
	}{
		{name: "exact answer", answer: "red", expected: true},
		{name: "different case", answer: "RED", expected: true},
		{name: "incorrect answer", answer: "blue", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quiz.IsAnswerCorrect(question, tt.answer); got != tt.expected {
				t.Errorf("IsAnswerCorrect(%q) = %t, want %t", tt.answer, got, tt.expected)
			}
		})
	}
}

func Test_QuizHandlerCreatesSession(t *testing.T) {
	store := &QuizStore{}
	handler := NewSessionQuizHandler(store, []quiz.Question{
		{Prompt: "Pick a color", CorrectAnswer: "red", Options: []string{"red", "blue"}},
	})

	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
	}{
		{name: "POST creates a session", method: http.MethodPost, target: "/quizzes", wantStatus: http.StatusSeeOther},
		{name: "GET root shows the start page", method: http.MethodGet, target: "/", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(tt.method, tt.target, nil))
			if response.Code != tt.wantStatus {
				t.Fatalf("%s %s status = %d, want %d", tt.method, tt.target, response.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusSeeOther {
				return
			}

			location := response.Header().Get("Location")
			id := strings.TrimPrefix(location, "/quizzes/")
			if _, ok := store.Get(id); !ok {
				t.Errorf("Location %q does not point at a stored session", location)
			}
		})
	}
}

func Test_QuizHandlerGetQuiz(t *testing.T) {
	tests := []struct {
		name       string
		questions  []quiz.Question
		wantStatus int
		wantBody   func(session *Session) []string
	}{
		{
			name: "displays the first question",
			questions: []quiz.Question{{
				Prompt:  "Pick a color",
				Options: []string{"red", "blue"},
			}},
			wantStatus: http.StatusOK,
			wantBody: func(session *Session) []string {
				return []string{
					"Pick a color",
					"<li>red</li>",
					"<li>blue</li>",
					`name="csrf_token" value="` + session.CSRFToken + `"`,
					`action="` + quizPath(session, "/answers") + `"`,
				}
			},
		},
		{
			name:       "rejects a quiz without questions",
			questions:  nil,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, session := newSessionTestHandler(t, tt.questions)
			response := getPage(handler, quizPath(session, ""))

			if response.Code != tt.wantStatus {
				t.Fatalf("GET quiz status = %d, want %d", response.Code, tt.wantStatus)
			}

			if tt.wantBody == nil {
				return
			}

			if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
				t.Errorf("Content-Type = %q, want text/html; charset=utf-8", got)
			}

			assertBody(t, "GET quiz", response.Body.String(), tt.wantBody(session), nil)
		})
	}
}

func Test_QuizHandlerAnswers(t *testing.T) {
	colorThenAnimal := []quiz.Question{
		{Prompt: "Pick a color", CorrectAnswer: "red", Options: []string{"red", "blue"}},
		{Prompt: "Pick an animal", Options: []string{"cat", "dog"}},
	}
	colorOnly := colorThenAnimal[:1]

	tests := []struct {
		name            string
		questions       []quiz.Question
		prior           []string // answers submitted with a valid token before the one under test
		answer          string
		validToken      bool
		wantStatus      int
		wantRedirect    bool
		wantResponse    func(session *Session) []string // POST response body
		wantPage        func(session *Session) []string // GET quiz body afterwards
		wantPageAbsent  func(session *Session) []string
		wantSubmissions int
	}{
		{
			name:         "correct answer in a different case advances",
			questions:    colorThenAnimal,
			answer:       "RED",
			validToken:   true,
			wantStatus:   http.StatusSeeOther,
			wantRedirect: true,
			wantPage: func(*Session) []string {
				return []string{"Pick an animal", "Phase: Correct", "well done"}
			},
			wantSubmissions: 1,
		},
		{
			name:         "incorrect answer on the last question completes the quiz",
			questions:    colorOnly,
			answer:       "blue",
			validToken:   true,
			wantStatus:   http.StatusSeeOther,
			wantRedirect: true,
			wantPage: func(session *Session) []string {
				return []string{
					"Quiz complete", "Phase: Incorrect", "Best luck on next", "Phase: Done",
					`action="` + quizPath(session, "/restart") + `"`,
				}
			},
			wantPageAbsent: func(session *Session) []string {
				return []string{`action="` + quizPath(session, "/answers") + `"`}
			},
			wantSubmissions: 1,
		},
		{
			name:         "answer after completion does not change the result",
			questions:    colorOnly,
			prior:        []string{"blue"},
			answer:       "red",
			validToken:   true,
			wantStatus:   http.StatusSeeOther,
			wantRedirect: true,
			wantPage: func(*Session) []string {
				return []string{"Phase: Incorrect"}
			},
			wantSubmissions: 1,
		},
		{
			name:       "answer outside the options is rejected",
			questions:  colorThenAnimal,
			answer:     "green",
			validToken: true,
			wantStatus: http.StatusBadRequest,
			wantResponse: func(session *Session) []string {
				return []string{
					"Pick a color",
					"<li>red</li>",
					"Please choose one of the listed options.",
					`action="` + quizPath(session, "/answers") + `"`,
				}
			},
			wantPage: func(*Session) []string {
				return []string{"Pick a color"}
			},
			wantSubmissions: 0,
		},
		{
			name:       "missing CSRF token is rejected",
			questions:  colorThenAnimal,
			answer:     "red",
			validToken: false,
			wantStatus: http.StatusForbidden,
			wantPage: func(*Session) []string {
				return []string{"Pick a color"}
			},
			wantSubmissions: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, session := newSessionTestHandler(t, tt.questions)
			for _, answer := range tt.prior {
				postAnswer(handler, session, answer, session.CSRFToken)
			}

			token := ""
			if tt.validToken {
				token = session.CSRFToken
			}

			response := postAnswer(handler, session, tt.answer, token)

			if response.Code != tt.wantStatus {
				t.Fatalf("POST answer status = %d, want %d", response.Code, tt.wantStatus)
			}

			if tt.wantRedirect {
				if got, want := response.Header().Get("Location"), quizPath(session, ""); got != want {
					t.Errorf("Location = %q, want %q", got, want)
				}
			}

			if tt.wantResponse != nil {
				assertBody(t, "POST answer", response.Body.String(), tt.wantResponse(session), nil)
			}

			// GET twice: a refresh after the redirect must not resubmit or skip a question.
			for i := 0; i < 2; i++ {
				var absent []string
				if tt.wantPageAbsent != nil {
					absent = tt.wantPageAbsent(session)
				}
				body := getPage(handler, quizPath(session, "")).Body.String()
				assertBody(t, "GET quiz", body, tt.wantPage(session), absent)
			}

			spec, _ := session.Quiz.Snapshot()
			if spec.SubmissionId != tt.wantSubmissions {
				t.Errorf("SubmissionId = %d, want %d", spec.SubmissionId, tt.wantSubmissions)
			}
		})
	}
}

func Test_QuizHandlerRestart(t *testing.T) {
	questions := []quiz.Question{{
		Prompt:        "Pick a color",
		CorrectAnswer: "red",
		Options:       []string{"red", "blue"},
	}}

	tests := []struct {
		name           string
		token          func(session *Session) string
		wantStatus     int
		wantRedirect   bool
		wantPage       func(session *Session) []string
		wantPageAbsent []string
	}{
		{
			name:         "valid token resets the quiz",
			token:        func(session *Session) string { return session.CSRFToken },
			wantStatus:   http.StatusSeeOther,
			wantRedirect: true,
			wantPage: func(session *Session) []string {
				return []string{"Pick a color", `action="` + quizPath(session, "/answers") + `"`}
			},
			wantPageAbsent: []string{"Phase:", "Restart quiz"},
		},
		{
			name:       "invalid token keeps the finished quiz",
			token:      func(*Session) string { return "invalid" },
			wantStatus: http.StatusForbidden,
			wantPage: func(*Session) []string {
				return []string{"Quiz complete", "Restart quiz"}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, session := newSessionTestHandler(t, questions)
			postAnswer(handler, session, "red", session.CSRFToken)

			if body := getPage(handler, quizPath(session, "")).Body.String(); !strings.Contains(body, "Restart quiz") {
				t.Fatalf("completion page should offer a restart button:\n%s", body)
			}

			response := postForm(handler, quizPath(session, "/restart"), url.Values{
				"csrf_token": {tt.token(session)},
			})
			if response.Code != tt.wantStatus {
				t.Fatalf("POST restart status = %d, want %d", response.Code, tt.wantStatus)
			}

			if tt.wantRedirect {
				if got, want := response.Header().Get("Location"), quizPath(session, ""); got != want {
					t.Errorf("Location = %q, want %q", got, want)
				}
			}

			body := getPage(handler, quizPath(session, "")).Body.String()
			assertBody(t, "GET quiz", body, tt.wantPage(session), tt.wantPageAbsent)
		})
	}
}

func Test_QuizHandlerRejectsCSRFTokenFromAnotherSession(t *testing.T) {
	questions := []quiz.Question{
		{Prompt: "Pick a color", CorrectAnswer: "red", Options: []string{"red", "blue"}},
		{Prompt: "Pick an animal", Options: []string{"cat", "dog"}},
	}

	tests := []struct {
		name       string
		path       string // suffix of the victim's quiz URL
		form       func(token string) url.Values
		useVictim  bool // use the victim's own token instead of the attacker's
		wantStatus int
		wantPrompt string // victim's prompt afterwards
	}{
		{
			name:       "answer with another session's token",
			path:       "/answers",
			form:       func(token string) url.Values { return url.Values{"answer": {"red"}, "csrf_token": {token}} },
			wantStatus: http.StatusForbidden,
			wantPrompt: "Pick a color",
		},
		{
			name:       "restart with another session's token",
			path:       "/restart",
			form:       func(token string) url.Values { return url.Values{"csrf_token": {token}} },
			wantStatus: http.StatusForbidden,
			wantPrompt: "Pick an animal",
		},
		{
			name:       "answer with its own token",
			path:       "/answers",
			form:       func(token string) url.Values { return url.Values{"answer": {"red"}, "csrf_token": {token}} },
			useVictim:  true,
			wantStatus: http.StatusSeeOther,
			wantPrompt: "Pick an animal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &QuizStore{}
			victim, err := store.NewSession(questions)
			if err != nil {
				t.Fatalf("NewSession() for victim error = %v", err)
			}

			attacker, err := store.NewSession(questions)
			if err != nil {
				t.Fatalf("NewSession() for attacker error = %v", err)
			}

			handler := NewSessionQuizHandler(store, questions)

			// The restart case needs progress to lose, so answer once first.
			if tt.path == "/restart" {
				postAnswer(handler, victim, "red", victim.CSRFToken)
			}

			token := attacker.CSRFToken
			if tt.useVictim {
				token = victim.CSRFToken
			}

			response := postForm(handler, quizPath(victim, tt.path), tt.form(token))

			if response.Code != tt.wantStatus {
				t.Errorf("POST %s status = %d, want %d", tt.path, response.Code, tt.wantStatus)
			}

			if body := getPage(handler, quizPath(victim, "")).Body.String(); !strings.Contains(body, tt.wantPrompt) {
				t.Errorf("victim quiz page does not contain %q after POST %s", tt.wantPrompt, tt.path)
			}
		})
	}
}

func Test_QuizHandlerRejectsUnknownSession(t *testing.T) {
	handler := NewSessionQuizHandler(&QuizStore{}, []quiz.Question{})
	response := getPage(handler, "/quizzes/unknown")

	if response.Code != http.StatusNotFound {
		t.Errorf("GET unknown quiz status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func newSessionTestHandler(t *testing.T, questions []quiz.Question) (http.Handler, *Session) {
	t.Helper()

	store := &QuizStore{}
	session, err := store.NewSession(questions)
	if err != nil {
		t.Fatalf("NewSession() error = %v", err)
	}

	return NewSessionQuizHandler(store, questions), session
}

func quizPath(session *Session, suffix string) string {
	return "/quizzes/" + session.ID + suffix
}

func postAnswer(handler http.Handler, session *Session, answer, csrfToken string) *httptest.ResponseRecorder {
	return postForm(handler, quizPath(session, "/answers"), url.Values{
		"answer":     {answer},
		"csrf_token": {csrfToken},
	})
}

func postForm(handler http.Handler, target string, values url.Values) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	return response
}

func getPage(handler http.Handler, target string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))

	return response
}

func assertBody(t *testing.T, label, body string, want, absent []string) {
	t.Helper()

	for _, w := range want {
		if !strings.Contains(body, w) {
			t.Errorf("%s body does not contain %q:\n%s", label, w, body)
		}
	}

	for _, a := range absent {
		if strings.Contains(body, a) {
			t.Errorf("%s body should not contain %q:\n%s", label, a, body)
		}
	}
}
