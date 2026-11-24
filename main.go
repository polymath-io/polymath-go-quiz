package main

import (
	"log"
	"net/http"
	"strings"
	"sync"
)

type Question struct {
	Prompt        string
	CorrectAnswer string
	Options       []string
}

type QuizSpec struct {
	Questions    []Question
	UserAnswer   string
	SubmissionId int
}

type QuizStatus struct {
	CurrentQuestion int
	Phase           string
	Feedback        string
	LastResult      string // feedback for the previous answer, shown on the next GET
}

type Quiz struct {
	mu     sync.Mutex
	Spec   QuizSpec
	Status QuizStatus
}

const (
	QuizPhasePending   string = "Pending"
	QuizPhaseCorrect   string = "Correct"
	QuizPhaseIncorrect string = "Incorrect"
	QuizPhaseDone      string = "Done"
)

func newQuestion(prompt string, correctAnswer string, options []string) Question {
	return Question{
		Prompt:        prompt,
		CorrectAnswer: correctAnswer,
		Options:       options,
	}
}

func isAnswerCorrect(question Question, userAns string) bool {
	if strings.ToLower(userAns) == strings.ToLower(question.CorrectAnswer) {
		return true
	}

	return false
}

func main() {
	q1 := newQuestion("first color?", "red", []string{"blue", "white", "red"})
	q2 := newQuestion("second color?", "blue", []string{"blue", "white", "red"})

	quiz := Quiz{
		Spec: QuizSpec{
			Questions:    []Question{q1, q2},
			UserAnswer:   "",
			SubmissionId: 0,
		},
		Status: QuizStatus{
			CurrentQuestion: 0,
			Phase:           QuizPhasePending,
			Feedback:        "",
		},
	}

	csrfToken, err := generateCSRFToken()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Server listening on http://localhost:8083")
	log.Fatal(http.ListenAndServe(":8083", newQuizHandler(&quiz, csrfToken)))
}
