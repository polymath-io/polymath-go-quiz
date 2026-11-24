package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
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
}

func showStatus(quiz Quiz) string {
	if quiz.Status.Phase == QuizPhasePending || quiz.Status.Phase == QuizPhaseDone {
		return fmt.Sprintf("Phase: %s", quiz.Status.Phase)
	}
	return fmt.Sprintf("Phase: %s\tMessage: %s", quiz.Status.Phase, quiz.Status.Feedback)

}

func renderQuizPage(w http.ResponseWriter, data PageData) {
	var body bytes.Buffer
	if err := pageTemplate.Execute(&body, data); err != nil {
		http.Error(w, "Could not render quiz page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(body.Bytes()); err != nil {
		log.Printf("Could not write quiz page: %v", err)
	}
}

func newQuizHandler(quiz Quiz) http.Handler {
	var mu sync.Mutex
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if quiz.Status.Phase == QuizPhaseDone {
			renderQuizPage(w, PageData{
				Prompt:      "Quiz complete",
				Messages:    []string{quiz.Status.LastResult, showStatus(quiz)},
				ShowRestart: true,
			})
			return
		}

		if quiz.Status.CurrentQuestion < 0 || quiz.Status.CurrentQuestion >= len(quiz.Spec.Questions) {
			http.Error(w, "No current quiz question", http.StatusInternalServerError)
			return
		}

		question := quiz.Spec.Questions[quiz.Status.CurrentQuestion]
		renderQuizPage(w, PageData{
			Prompt:   question.Prompt,
			Options:  question.Options,
			Messages: []string{quiz.Status.LastResult},
			ShowForm: true,
		})
	})

	mux.HandleFunc("POST /{$}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if quiz.Status.Phase == QuizPhaseDone {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if quiz.Status.CurrentQuestion < 0 || quiz.Status.CurrentQuestion >= len(quiz.Spec.Questions) {
			http.Error(w, "No current quiz question", http.StatusInternalServerError)
			return
		}

		question := quiz.Spec.Questions[quiz.Status.CurrentQuestion]
		received := r.FormValue("answer")

		quiz.Spec.UserAnswer = received
		quiz.Spec.SubmissionId++

		if isAnswerCorrect(question, received) {
			quiz.Status.Phase = QuizPhaseCorrect
			quiz.Status.Feedback = "well done"
		} else {
			quiz.Status.Phase = QuizPhaseIncorrect
			quiz.Status.Feedback = "Best luck on next"
		}
		quiz.Status.LastResult = showStatus(quiz) // keep Correct/Incorrect before moving on

		if quiz.Status.CurrentQuestion+1 < len(quiz.Spec.Questions) {
			quiz.Status.CurrentQuestion++
			quiz.Status.Phase = QuizPhasePending
		} else {
			quiz.Status.Phase = QuizPhaseDone
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	mux.HandleFunc("POST /restart", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		quiz.Spec.UserAnswer = ""
		quiz.Spec.SubmissionId = 0
		quiz.Status = QuizStatus{Phase: QuizPhasePending}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	return mux
}
