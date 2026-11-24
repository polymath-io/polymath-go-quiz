package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

//go:embed index.html
var pageHTML string

var pageTemplate = template.Must(template.New("page").Parse(pageHTML))

type PageData struct {
	Prompt    string
	Options   []string
	Processed string
	ShowForm  bool
}

func showStatus(quiz Quiz) string {
	if quiz.Status.Phase == QuizPhasePending || quiz.Status.Phase == QuizPhaseDone {
		return fmt.Sprintf("Phase: %s\n", quiz.Status.Phase)
	}
	return fmt.Sprintf("Phase: %s\tMessage: %s\n", quiz.Status.Phase, quiz.Status.Feedback)

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
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if quiz.Status.Phase == QuizPhaseDone {
			renderQuizPage(w, PageData{
				Prompt:    "Quiz complete",
				Processed: showStatus(quiz),
			})
			return
		}

		if quiz.Status.CurrentQuestion < 0 || quiz.Status.CurrentQuestion >= len(quiz.Spec.Questions) {
			http.Error(w, "No current quiz question", http.StatusInternalServerError)
			return
		}

		question := quiz.Spec.Questions[quiz.Status.CurrentQuestion]
		data := PageData{
			Prompt:   question.Prompt,
			Options:  question.Options,
			ShowForm: true,
		}
		renderQuizPage(w, data)
	})

	mux.HandleFunc("POST /{$}", func(w http.ResponseWriter, r *http.Request) {
		if quiz.Status.CurrentQuestion < 0 || quiz.Status.CurrentQuestion >= len(quiz.Spec.Questions) {
			http.Error(w, "No current quiz question", http.StatusInternalServerError)
			return
		}

		question := quiz.Spec.Questions[quiz.Status.CurrentQuestion]
		received := r.FormValue("answer")

		if isAnswerCorrect(question, received) {
			quiz.Status.Feedback = "well done"
			quiz.Status.Phase = QuizPhaseCorrect
		} else {
			quiz.Status.Phase = QuizPhaseIncorrect
			quiz.Status.Feedback = "Best luck on next"
		}

		feedback := showStatus(quiz) // save Correct/Incorrect before changing the phase

		if quiz.Status.CurrentQuestion+1 < len(quiz.Spec.Questions) {
			quiz.Status.CurrentQuestion++
			quiz.Status.Phase = QuizPhasePending

			next := quiz.Spec.Questions[quiz.Status.CurrentQuestion]
			data := PageData{
				Prompt:    next.Prompt,
				Options:   next.Options,
				Processed: feedback,
				ShowForm:  true,
			}
			renderQuizPage(w, data)
		} else {
			quiz.Status.Phase = QuizPhaseDone
			data := PageData{
				Prompt:    "Quiz complete",
				Processed: feedback + showStatus(quiz),
			}
			renderQuizPage(w, data)
		}
	})

	return mux
}
