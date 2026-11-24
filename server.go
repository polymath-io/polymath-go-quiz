package main

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
	"strings"
)

//go:embed index.html
var pageHTML string

var pageTemplate = template.Must(template.New("page").Parse(pageHTML))

type PageData struct {
	Prompt      string
	Options     []string
	Messages    []string
	ShowForm    bool
	ShowRestart bool
	CSRFToken   string
}

func showStatus(status QuizStatus) string {
	if status.Phase == QuizPhasePending || status.Phase == QuizPhaseDone {
		return fmt.Sprintf("Phase: %s", status.Phase)
	}
	return fmt.Sprintf("Phase: %s\tMessage: %s", status.Phase, status.Feedback)

}

func generateCSRFToken() (string, error) {
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

func isAnswerOption(options []string, answer string) bool {
	for _, option := range options {
		if strings.EqualFold(option, answer) {
			return true
		}
	}
	return false
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

func newQuizHandler(quiz *Quiz, csrfToken string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		quiz.mu.Lock()
		defer quiz.mu.Unlock()

		if quiz.Status.Phase == QuizPhaseDone {
			renderQuizPage(w, PageData{
				Prompt:      "Quiz complete",
				Messages:    []string{quiz.Status.LastResult, showStatus(quiz.Status)},
				ShowRestart: true,
				CSRFToken:   csrfToken,
			})
			return
		}

		if quiz.Status.CurrentQuestion < 0 || quiz.Status.CurrentQuestion >= len(quiz.Spec.Questions) {
			http.Error(w, "No current quiz question", http.StatusInternalServerError)
			return
		}

		question := quiz.Spec.Questions[quiz.Status.CurrentQuestion]
		renderQuizPage(w, PageData{
			Prompt:    question.Prompt,
			Options:   question.Options,
			Messages:  []string{quiz.Status.LastResult},
			ShowForm:  true,
			CSRFToken: csrfToken,
		})
	})

	mux.HandleFunc("POST /{$}", func(w http.ResponseWriter, r *http.Request) {
		if !requireValidCSRFToken(w, r, csrfToken) {
			return
		}

		quiz.mu.Lock()
		defer quiz.mu.Unlock()

		if quiz.Status.Phase == QuizPhaseDone {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if quiz.Status.CurrentQuestion < 0 || quiz.Status.CurrentQuestion >= len(quiz.Spec.Questions) {
			http.Error(w, "No current quiz question", http.StatusInternalServerError)
			return
		}

		question := quiz.Spec.Questions[quiz.Status.CurrentQuestion]
		received := r.Form.Get("answer")
		if !isAnswerOption(question.Options, received) {
			renderQuizPageWithStatus(w, http.StatusBadRequest, PageData{
				Prompt:    question.Prompt,
				Options:   question.Options,
				Messages:  []string{"Please choose one of the listed options.", quiz.Status.LastResult},
				ShowForm:  true,
				CSRFToken: csrfToken,
			})
			return
		}

		quiz.Spec.UserAnswer = received
		quiz.Spec.SubmissionId++

		if isAnswerCorrect(question, received) {
			quiz.Status.Phase = QuizPhaseCorrect
			quiz.Status.Feedback = "well done"
		} else {
			quiz.Status.Phase = QuizPhaseIncorrect
			quiz.Status.Feedback = "Best luck on next"
		}
		quiz.Status.LastResult = showStatus(quiz.Status) // keep Correct/Incorrect before moving on

		if quiz.Status.CurrentQuestion+1 < len(quiz.Spec.Questions) {
			quiz.Status.CurrentQuestion++
			quiz.Status.Phase = QuizPhasePending
		} else {
			quiz.Status.Phase = QuizPhaseDone
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	mux.HandleFunc("POST /restart", func(w http.ResponseWriter, r *http.Request) {
		if !requireValidCSRFToken(w, r, csrfToken) {
			return
		}

		quiz.mu.Lock()
		defer quiz.mu.Unlock()

		quiz.Spec.UserAnswer = ""
		quiz.Spec.SubmissionId = 0
		quiz.Status = QuizStatus{Phase: QuizPhasePending}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	return mux
}
