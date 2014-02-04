package greyhound

import (
	"bufio"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// A PhpHandler represents a single PHP process running the built-in Web server.
//
// Due to the need to check for errors in the STDERR of the process it only allows one call to ServeHTTP() at a time (using sync.Mutex).
type PhpHandler struct {
	dir        string
	port       int
	host       string
	cmd        *exec.Cmd
	stderr     *bufio.Reader
	errorLog   chan string
	requestLog chan string
	errorChan  chan error
	mutex      *sync.Mutex
	timeout    time.Duration
	ignore     []string
}

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

// NewPhpHandler starts a new PHP server listening on the first free port (between port 8001 and 2^16).
//
// Usage:
// 	ph, err := NewPhpHandler("/path/to/web/root", 1000)
// 	if err != nil {
// 	        panic(err)
// 	}
// 	defer ph.Close()
//
// timeout is in milliseconds
func NewPhpHandler(dir string, timeout time.Duration, ignore []string) (ph *PhpHandler, err error) {
	for p := 8001; p < int(math.Pow(2, 16)); p++ {
		ph = &PhpHandler{
			dir: dir,
			// Use 127.0.0.1 here instead of localhost
			// otherwise PHP only listens on ::1
			host: fmt.Sprintf("127.0.0.1:%d", p),
		}
		cmd, stderr, errorChan, err := runPhp(ph.dir, ph.host)

		if err == nil {
			ph.timeout = timeout
			ph.cmd = cmd
			ph.stderr = bufio.NewReader(stderr)
			ph.errorLog = make(chan string)
			ph.requestLog = make(chan string)
			ph.errorChan = errorChan
			ph.mutex = &sync.Mutex{}
			ph.ignore = ignore
			go ph.listenForErrors()
			return ph, nil
		}
	}
	return nil, errors.New("no free ports found")
}

// Close must be called after a successful call to NewPhpHandler otherwise you may get stray PHP processes floating around.
func (ph *PhpHandler) Close() {
	err := ph.cmd.Process.Kill()
	if err != nil {
		panic(err)
	}
}

// ServeHTTP sends an http.Request to the PHP process, writes what it gets to an http.ResponseWriter.
//
// If an error gets printed to STDERR during the request, it shows the error instead of what PHP returned. If the request takes too long it shows a message saying that the request took too long (see timeout option on NewPhpHandler).
func (ph *PhpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ph.mutex.Lock()
	defer ph.mutex.Unlock()

	var err error

	r.URL.Scheme = "http"
	r.URL.Host = ph.host

	// Make the request
	tr := &http.Transport{}

	// Timeout stuff
	var resp *http.Response
	wait := make(chan bool)

	go func() {
		resp, err = tr.RoundTrip(r)
		wait <- true
	}()

	select {
	case <-wait:
	case <-time.After(ph.timeout):
		renderError(w, "timeoutError", ph.timeout.String())
		return
	}
	// End timeout stuff

	if err != nil {
		renderError(w, "requestError", "uh oh")
		return
	}
	defer resp.Body.Close()

	// The request gets printed to STDERR only after the errors
	// So it's a reliable way to confirm that the page was returned

FOR:
	for {
		select {
		case <-ph.errorChan:
			renderError(w, "earlyExit", "oh dear")
			return
		case <-ph.requestLog:
			break FOR
		case line := <-ph.errorLog:
			ignoreError := false
		IGNORE:
			for _, i := range ph.ignore {
				if strings.Contains(line, i) {
					ignoreError = true
					break IGNORE
				}
			}

			if !ignoreError {
				renderError(w, "interpreterError", line)
				ph.resetErrors()
				return
			}
		}
	}

	// Headers
	headers := w.Header()
	for k, v := range resp.Header {
		headers[k] = v
	}

	// Status code
	w.WriteHeader(resp.StatusCode)

	// Body
	bufWriter := bufio.NewWriter(w)
	bufWriter.ReadFrom(resp.Body)
	bufWriter.Flush()

	return
}

// Converts bufio.Reader into chan for ease of use during the request
func (ph *PhpHandler) listenForErrors() {
	for {
		line, err := ph.stderr.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				return
			}
			panic(err)
		}

		if line[25:37] != "] 127.0.0.1:" {
			ph.errorLog <- line[40:]
		} else {
			ph.requestLog <- line[38:]
		}
	}
}

// Consumes all the errors until the request completes and then returns
func (ph *PhpHandler) resetErrors() {
	for {
		select {
		case <-ph.errorLog:
			// consume the error
		case <-ph.requestLog:
			return
		}
	}
}

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

// A low-level command
// Starts PHP running, waits half a second, returns an error if PHP stopped during that time
func runPhp(dir string, host string) (cmd *exec.Cmd, stderr io.ReadCloser, errorChan chan error, err error) {
	cmd = exec.Command(
		"php",
		"-n", // do not read php.ini
		"-S", host,
		"-t", dir,
		"-d", "display_errors=Off",
		"-d", "log_errors=On",
		"-d", "error_reporting=E_ALL",
	)

	// Connect stderr
	stderr, err = cmd.StderrPipe()
	if err != nil {
		return
	}

	// Let's go
	err = cmd.Start()
	if err != nil {
		return
	}

	// Wait 1 second for the command to terminate
	// If it exits early, that's bad whatever the exit status
	errorChan = make(chan error)

	go func() {
		err := cmd.Wait()
		if err != nil {
			errorChan <- err
		} else {
			errorChan <- errors.New("command exited early")
		}
	}()

	select {
	case <-time.After(time.Millisecond * 500):
		return
	case err = <-errorChan:
		return
	}
}
