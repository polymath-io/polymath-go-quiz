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

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write(startHTML); err != nil {
			log.Printf("Could not write start page: %v", err)
		}
	})

	// Create a new session and send the visitor to it.
	mux.HandleFunc("POST /quizzes", func(w http.ResponseWriter, r *http.Request) {
		session, err := store.NewSession(questions)
		if err != nil {
			http.Error(w, "Could not create quiz", http.StatusInternalServerError)

			return
		}

		http.Redirect(w, r, "/quizzes/"+session.ID, http.StatusSeeOther)
	})

	dispatch := func(w http.ResponseWriter, r *http.Request) {
		session, ok := store.Get(r.PathValue("id"))
		if !ok {
			http.NotFound(w, r)

			return
		}

		newQuizHandler(session.Quiz, session.CSRFToken, session.ID).ServeHTTP(w, r)
	}

	mux.HandleFunc("GET /quizzes/{id}", dispatch)
	mux.HandleFunc("POST /quizzes/{id}/answers", dispatch)
	mux.HandleFunc("POST /quizzes/{id}/restart", dispatch)

	return mux
}

func newQuizHandler(q *quiz.Quiz, csrfToken, sessionID string) http.Handler {
	mux := http.NewServeMux()
	quizURL := "/quizzes/" + sessionID
	answerURL := quizURL + "/answers"
	restartURL := quizURL + "/restart"

	mux.HandleFunc("GET /quizzes/{id}", func(w http.ResponseWriter, r *http.Request) {
		spec, status := q.Snapshot()

		if status.Phase == quiz.QuizPhaseDone {
			renderQuizPage(w, PageData{
				Prompt:      "Quiz complete",
				Messages:    []string{status.LastResult, quiz.ShowStatus(status)},
				ShowRestart: true,
				CSRFToken:   csrfToken,
				RestartURL:  restartURL,
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
			CSRFToken: csrfToken,
			AnswerURL: answerURL,
		})
	})

	mux.HandleFunc("POST /quizzes/{id}/answers", func(w http.ResponseWriter, r *http.Request) {
		if !requireValidCSRFToken(w, r, csrfToken) {
			return
		}

		outcome := q.SubmitAnswer(r.Form.Get("answer"))
		switch outcome.Result {
		case quiz.SubmitNoQuestion:
			http.Error(w, "No current quiz question", http.StatusInternalServerError)
		case quiz.SubmitInvalidOption:
			renderQuizPageWithStatus(w, http.StatusBadRequest, PageData{
				Prompt:    outcome.Question.Prompt,
				Options:   outcome.Question.Options,
				Messages:  []string{"Please choose one of the listed options.", outcome.LastResult},
				ShowForm:  true,
				CSRFToken: csrfToken,
				AnswerURL: answerURL,
			})
		default:
			http.Redirect(w, r, quizURL, http.StatusSeeOther)
		}
	})

	mux.HandleFunc("POST /quizzes/{id}/restart", func(w http.ResponseWriter, r *http.Request) {
		if !requireValidCSRFToken(w, r, csrfToken) {
			return
		}
		q.Restart()
		http.Redirect(w, r, quizURL, http.StatusSeeOther)
	})

	return mux
}
