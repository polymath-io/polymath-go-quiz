package main

import (
	_ "embed"
	"fmt"
	"html/template"

	"net/http"
)



//go:embed index.html
var pageHTML string

var tmpl = template.Must(template.New("page").Parse(pageHTML))

type PageData struct {
	Input     string
	Processed string
}

func processInput(text string) string {
	// do something
	processed := text + text
	return fmt.Sprintf(processed)
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	if err := tmpl.Execute(w, PageData{}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	received := r.FormValue("userInput")

	data := PageData{
		Input:     received,               // sending back the input
		Processed: processInput(received), // sending the process input
	}

	w.Header().Set("Refresh", "3; url=/")

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
