package web

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"quiz.com/internal/quiz"
)

//go:embed index.html
var pageHTML string

var pageTemplate = template.Must(template.New("page").Parse(pageHTML))

//go:embed start.html
var startHTML []byte

type PageData struct {
	Prompt      string
	Options     []string
	Messages    []string
	ShowForm    bool
	ShowRestart bool
	CSRFToken   string
	AnswerURL   string
	RestartURL  string
}

func GenerateCSRFToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", fmt.Errorf("generate CSRF token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(token), nil
}

func requireValidCSRFToken(w http.ResponseWriter, r *http.Request, expected string) bool {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return false
	}
	if subtle.ConstantTimeCompare([]byte(r.Form.Get("csrf_token")), []byte(expected)) != 1 {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return false
	}
	return true
}

func renderQuizPage(w http.ResponseWriter, data PageData) {
	renderQuizPageWithStatus(w, http.StatusOK, data)
}

func renderQuizPageWithStatus(w http.ResponseWriter, status int, data PageData) {
	var body bytes.Buffer
	if err := pageTemplate.Execute(&body, data); err != nil {
		http.Error(w, "Could not render quiz page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(body.Bytes()); err != nil {
		log.Printf("Could not write quiz page: %v", err)
	}
}

func NewSessionQuizHandler(store *QuizStore, questions []quiz.Question) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", serveStartPage)
	mux.HandleFunc("POST /quizzes", createSession(store, questions))

	mux.HandleFunc("GET /quizzes/{id}", withSession(store, showQuiz))
	mux.HandleFunc("POST /quizzes/{id}/answers", withSession(store, submitAnswer))
	mux.HandleFunc("POST /quizzes/{id}/restart", withSession(store, restartQuiz))

	return mux
}

func quizURL(session *Session) string {
	return "/quizzes/" + session.ID
}

// withSession looks up the session named by the {id} path value once and passes
// it to h, or responds with 404 if there is no such (unexpired) session.
func withSession(store *QuizStore, h func(http.ResponseWriter, *http.Request, *Session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, ok := store.Get(r.PathValue("id"))
		if !ok {
			http.NotFound(w, r)

			return
		}

		h(w, r, session)
	}
}

func serveStartPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(startHTML); err != nil {
		log.Printf("Could not write start page: %v", err)
	}
}

// createSession starts a new quiz session and sends the visitor to it.
func createSession(store *QuizStore, questions []quiz.Question) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := store.NewSession(questions)
		if err != nil {
			http.Error(w, "Could not create quiz", http.StatusInternalServerError)

			return
		}

		http.Redirect(w, r, quizURL(session), http.StatusSeeOther)
	}
}

func showQuiz(w http.ResponseWriter, r *http.Request, session *Session) {
	spec, status := session.Quiz.Snapshot()

	if status.Phase == quiz.QuizPhaseDone {
		renderQuizPage(w, PageData{
			Prompt:      "Quiz complete",
			Messages:    []string{status.LastResult, quiz.ShowStatus(status)},
			ShowRestart: true,
			CSRFToken:   session.CSRFToken,
			RestartURL:  quizURL(session) + "/restart",
		})
		return
	}

	if status.CurrentQuestion < 0 || status.CurrentQuestion >= len(spec.Questions) {
		http.Error(w, "No current quiz question", http.StatusInternalServerError)
		return
	}

	question := spec.Questions[status.CurrentQuestion]
	renderQuizPage(w, PageData{
		Prompt:    question.Prompt,
		Options:   question.Options,
		Messages:  []string{status.LastResult},
		ShowForm:  true,
		CSRFToken: session.CSRFToken,
		AnswerURL: quizURL(session) + "/answers",
	})
}

func submitAnswer(w http.ResponseWriter, r *http.Request, session *Session) {
	if !requireValidCSRFToken(w, r, session.CSRFToken) {
		return
	}

	outcome := session.Quiz.SubmitAnswer(r.Form.Get("answer"))
	switch outcome.Result {
	case quiz.SubmitNoQuestion:
		http.Error(w, "No current quiz question", http.StatusInternalServerError)
	case quiz.SubmitInvalidOption:
		renderQuizPageWithStatus(w, http.StatusBadRequest, PageData{
			Prompt:    outcome.Question.Prompt,
			Options:   outcome.Question.Options,
			Messages:  []string{"Please choose one of the listed options.", outcome.LastResult},
			ShowForm:  true,
			CSRFToken: session.CSRFToken,
			AnswerURL: quizURL(session) + "/answers",
		})
	default:
		http.Redirect(w, r, quizURL(session), http.StatusSeeOther)
	}
}

func restartQuiz(w http.ResponseWriter, r *http.Request, session *Session) {
	if !requireValidCSRFToken(w, r, session.CSRFToken) {
		return
	}

	session.Quiz.Restart()
	http.Redirect(w, r, quizURL(session), http.StatusSeeOther)
}
