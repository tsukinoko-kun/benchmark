package main

import (
	"fmt"
	"github.com/tsukinoko-kun/benchmark/run"
	"io"
	"net/http"
	"os"
)

const indexHtml = `<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Benchmark</title>
	</head>
	<body>
		<h1>Benchmark</h1>
		<form action="/submit" target="_self" method="post" enctype="multipart/form-data">
			<input name="source" id="source" type="file" required accept=".java">
			<select name="language" id="language" required onchange="document.getElementById('source').accept = '.' + this.value">
				<option value="java" selected>Java</option>
				<option value="go">Go</option>
			</select>
			<input type="submit" value="Submit">
		</form>
	</body>
</html>`

func indexHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprint(w, indexHtml)
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	file, _, err := r.FormFile("source")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return
	}
	defer file.Close()

	// Read the file
	code, err := io.ReadAll(file)
	if err != nil {
		_, _ = fmt.Fprintln(w, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	language := r.FormValue("language")
	if language == "" {
		_, _ = fmt.Fprintln(w, "language is required")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Run the code
	output, err := run.Run(code, language)
	if err != nil {
		_, _ = fmt.Fprintln(w, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, _ = fmt.Fprint(w, string(output))
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/submit", submitHandler)
	if err := http.ListenAndServe(":80", nil); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}
