package greyhound

import (
	"html/template"
	"net/http"
)

type phpError struct {
	ErrorType string
	Text      string
}

var tmpl = template.Must(template.New("").Parse(`
{{define "interpreterError"}}
	<h1>Error</h1>
	<div class="error">
		<p><code>{{.}}</code></p>
	</div>
{{end}}

{{define "timeoutError"}}
	<h1>Timeout error</h1>
	<div class="error">
		<p>Waited {{.}} and received no response</p>
	</div>
{{end}}

{{define "otherError"}}
	<h1>Other error</h1>
	<div class="error">
		<p>{{.}}</p>
	</div>
{{end}}

<!doctype html>
<html>
	<head>
		<title>Error</title>
		<style>
			html {
				padding: 0;
			}
			body {
				margin: 0;
				color: black;
				background: white;
				font-family: sans-serif;
			}
			.container {
				max-width: 700px;
				background: #ddd;
				margin: 0 auto;
				padding: 20px;
			}
			h1 {
				font-weight: normal;
			}
			.error {
				overflow-x: auto;
				background: #fdd;
			}
		</style>
	</head>
	<body>

		<div class="container">

			{{if eq .ErrorType "interpreterError"}}
				{{template "interpreterError" .Text}}
			{{else if eq .ErrorType "timeoutError"}}
				{{template "timeoutError" .Text}}
			{{else}}
				{{template "otherError" .Text}}
			{{end}}

		</div>
	</body>
</html>
`))

// Render the error template
func renderError(w http.ResponseWriter, t string, s string) {
	w.WriteHeader(http.StatusInternalServerError)

	e := phpError{
		ErrorType: t,
		Text:      s,
	}

	err := tmpl.Execute(w, e)
	if err != nil {
		panic(err)
	}
}
