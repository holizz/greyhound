package greyhound

import (
	"html/template"
	"log"
	"net/http"
)

type phpError struct {
	ErrorType string
	Text      string
}

var tmpl = template.Must(template.New("").Parse(
	`<!doctype html>
	<title>Error</title>

	{{if eq .ErrorType "interpreterError"}}

		<h1>Error</h1>
		<pre>{{.Text}}</pre>

	{{else if eq .ErrorType "timeoutError"}}

		<h1>Timeout error</h1>
		<p>Waited {{.Text}} and received no response</p>

	{{else}}

		<h1>Request error</h1>
		<p>{{.Text}}</p>

	{{end}}
	`,
))

// Render the error template
func renderError(w http.ResponseWriter, t string, s string) {
	w.WriteHeader(http.StatusInternalServerError)

	e := phpError{
		ErrorType: t,
		Text:      s,
	}

	err := tmpl.Execute(w, e)
	if err != nil {
		log.Fatalln("Template failed to execute")
	}
}
