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

	var store web.QuizStore

	handler := web.NewSessionQuizHandler(&store, []quiz.Question{q1, q2})

	log.Println("Server listening on http://localhost:8083")
	log.Fatal(http.ListenAndServe(":8083", handler))
}
