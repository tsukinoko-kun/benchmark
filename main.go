package main

import (
	"encoding/json"
	"fmt"
	"github.com/tsukinoko-kun/benchmark/run"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"strings"

	"github.com/gorilla/websocket"
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
		<select id="language" required onchange="document.getElementById('source').accept = '.' + this.value">
			<option value="java" selected>Java</option>
			<option value="go">Go</option>
		</select>
		<input id="source" type="file" required accept=".java">
		<br>
		<button onclick="submit()">Submit</button>
		<br>
		<pre><code id="output">Select a language and a file, then click Submit.</code></pre>
		<script>
			const sourceEl = document.getElementById('source');
			const languageEl = document.getElementById('language');
			const outputEl = document.getElementById('output');

			function submit() {
				console.log('submit');
				outputEl.textContent = 'Running...';
				outputEl.classList.remove('error');
				try {
					const source = sourceEl.files[0];
					const language = languageEl.value;
					if (!source) {
						outputEl.textContent = 'Please select a file';
						outputEl.classList.add('error');
						return;
					}

					const reader = new FileReader();
					reader.onload = function() {
						const ws = new WebSocket('ws://' + window.location.host + '/ws');
						ws.addEventListener('open', function() {
							ws.send(JSON.stringify({
								code: reader.result,
								language,
							}));
						});
						ws.addEventListener('message', function(event) {
							const data = JSON.parse(event.data);
							console.log(data);
							if (data.error) {
								outputEl.textContent = data.error;
								outputEl.classList.add('error');
							} else {
								outputEl.textContent = data.output;
								outputEl.classList.remove('error');
							}
						});
						ws.addEventListener('error', function(ev) {
							outputEl.textContent = "WebSocket error occurred";
							if (ev.message) {
								outputEl.textContent += ": " + ev.message;
							}
							outputEl.classList.add('error');
						});
					};
					reader.onerror = function() {
						outputEl.textContent = 'Failed to read the file';
						outputEl.classList.add('error');
					};
					reader.readAsText(source);
				} catch (error) {
					outputEl.textContent = error.message;
					outputEl.classList.add('error');
				}
			}
		</script>
		<style>
			:root {
				color-scheme: light dark;
			}
			.error {
				color: red;
			}
		</style>
	</body>
</html>`

func indexHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprint(w, indexHtml)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func init() {
	var origins []string
	if originsEnv, ok := os.LookupEnv("ORIGINS"); ok {
		origins = strings.Split(originsEnv, ";")
	} else if originEnv, ok := os.LookupEnv("ORIGIN"); ok {
		origins = []string{originEnv}
	} else {
		return
	}

	upgrader.CheckOrigin = func(r *http.Request) bool {
		for _, o := range origins {
			ro := r.Header.Get("Origin")
			// check string equality
			if ro == o {
				return true
			}
			// check glob pattern
			if match, _ := path.Match(o, ro); match {
				return true
			}
		}
		return false
	}
}

type (
	wsRequest struct {
		Language string `json:"language"`
		Code     string `json:"code"`
	}
	wsResponse struct {
		Output string `json:"output"`
	}
	wsError struct {
		Error string `json:"error"`
	}
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Only GET requests are allowed", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	_, message, err := conn.ReadMessage()
	if err != nil {
		return
	}
	log.Println("received message")

	// Parse the message
	var msg wsRequest
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Println(err)
		return
	}

	// Run the code
	output, err := run.Run([]byte(msg.Code), msg.Language)
	// Send the output
	if err != nil {
		_ = conn.WriteJSON(wsError{Error: err.Error()})
		return
	} else {
		_ = conn.WriteJSON(wsResponse{Output: string(output)})
	}
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/ws", wsHandler)
	if err := http.ListenAndServe(":80", nil); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}
