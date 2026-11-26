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

	session, err := store.NewSession([]quiz.Question{q1, q2})
	if err != nil {
		log.Fatal(err)
	}

	quizURL := "/quizzes/" + session.ID

	mux := http.NewServeMux()
	mux.Handle("GET /{$}", http.RedirectHandler(quizURL, http.StatusSeeOther))
	mux.Handle("/", web.NewSessionQuizHandler(&store))

	log.Printf("Quiz session: http://localhost:8083%s", quizURL)
	log.Println("Server listening on http://localhost:8083")
	log.Fatal(http.ListenAndServe(":8083", mux))
}
