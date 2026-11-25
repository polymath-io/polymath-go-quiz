package main

import (
	"log"
	"net/http"

	"quiz.com/internal/quiz"
	"quiz.com/internal/web"
)

func main() {
	q1 := quiz.NewQuestion("first color?", "red", []string{"blue", "white", "red"})
	q2 := quiz.NewQuestion("second color?", "blue", []string{"blue", "white", "red"})
	currentQuiz := quiz.New([]quiz.Question{q1, q2})

	csrfToken, err := web.GenerateCSRFToken()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Server listening on http://localhost:8083")
	log.Fatal(http.ListenAndServe(":8083", web.NewQuizHandler(currentQuiz, csrfToken)))
}
