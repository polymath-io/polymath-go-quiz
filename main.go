package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
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
}

type Quiz struct {
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

func getAnswer(question Question, input *bufio.Scanner) (string, error) {
	fmt.Println(question.Prompt)
	fmt.Println("Options:")
	for _, option := range question.Options {
		fmt.Printf("  - %s\n", option)
	}
	fmt.Print("> ")
	if !input.Scan() {
		if err := input.Err(); err != nil {
			return "", err
		}

		return "", io.EOF
	}

	return strings.TrimSpace(input.Text()), nil
}

func isAnswerCorrect(question Question, userAns string) bool {
	if strings.ToLower(userAns) == strings.ToLower(question.CorrectAnswer) {
		return true
	}

	return false
}

func showStatus(quiz Quiz) {
	if quiz.Status.Phase == QuizPhasePending || quiz.Status.Phase == QuizPhaseDone {
		fmt.Printf("Phase: %s\n", quiz.Status.Phase)
	} else {
		fmt.Printf("Phase: %s\tMessage: %s\n", quiz.Status.Phase, quiz.Status.Feedback)
	}

}

func main() {

	input := bufio.NewScanner(os.Stdin)

	q1 := newQuestion("first color?", "red", []string{"blue", "white", "red"})
	q2 := newQuestion("second color?", "blue", []string{"blue", "white", "red"})

	// create a Quiz
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

	for _, q := range quiz.Spec.Questions {
		showStatus(quiz)
		quiz.Status.CurrentQuestion++

		// prompt question and get answer
		answer, err := getAnswer(q, input)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Could not read answer:", err)
			return
		}

		quiz.Spec.UserAnswer = answer
		quiz.Spec.SubmissionId++

		if isAnswerCorrect(q, answer) {
			quiz.Status.Feedback = "well done"
			quiz.Status.Phase = QuizPhaseCorrect
		} else {
			quiz.Status.Phase = QuizPhaseIncorrect
			quiz.Status.Feedback = "Best luck on next"
		}

		showStatus(quiz)

		time.Sleep(time.Second * 1)
		if quiz.Status.CurrentQuestion == len(quiz.Spec.Questions) {
			quiz.Status.Phase = QuizPhaseDone
			showStatus(quiz)
		} else {
			quiz.Status.Phase = QuizPhasePending
		}
	}

}
