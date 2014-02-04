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
	"strconv"
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
	stdout     chan string
	stderr     chan string
	errorLog   chan string
	requestLog chan string
	mutex      *sync.Mutex
	timeout    int
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
		<p>Waited {{.Text}}ms and received no response</p>

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
func NewPhpHandler(dir string, timeout int) (ph *PhpHandler, err error) {
	for p := 8001; p < int(math.Pow(2, 16)); p++ {
		ph = &PhpHandler{
			dir: dir,
			// Use 127.0.0.1 here instead of localhost
			// otherwise PHP only listens on ::1
			host: fmt.Sprintf("127.0.0.1:%d", p),
		}
		cmd, stdout, stderr, err := runPhp(ph.dir, ph.host)

		if err == nil {
			ph.timeout = timeout
			ph.cmd = cmd
			ph.stdout = stdout
			ph.stderr = stderr
			ph.errorLog = make(chan string)
			ph.requestLog = make(chan string)
			ph.mutex = &sync.Mutex{}
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
	case <-time.After(time.Millisecond * time.Duration(ph.timeout)):
		renderError(w, "timeoutError", strconv.Itoa(ph.timeout))
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
		case <-ph.requestLog:
			break FOR
		case line := <-ph.errorLog:
			renderError(w, "interpreterError", line)
			ph.resetErrors()
			return
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

// Converts stderr into two different chans for errors and request logs
func (ph *PhpHandler) listenForErrors() {
	for {
		line := <-ph.stderr

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

// Starts PHP running
func runPhp(dir string, host string) (cmd *exec.Cmd, stdout, stderr chan string, err error) {
	fmt.Printf("runPhp: %s\n", host)
	cmd, stdout, stderr = makeCmd([]string{
		"php",
		"-n", // do not read php.ini
		"-S", host,
		"-t", dir,
		"-d", "display_errors=Off",
		"-d", "log_errors=On",
		"-d", "error_reporting=E_ALL",
	})

	fmt.Println("runPhp: starting...")
	err = cmd.Start()
	if err != nil {
		panic(err)
	}
	fmt.Println(<-stdout)
	fmt.Println("runPhp: done")

	timeout := time.After(time.Millisecond * 1000)
	started := make(chan bool)

	go func() {
		for {
			line := <-stdout
			fmt.Printf("runPhp: line: %s\n", line)
			if strings.HasPrefix(line, "Listening on http://") {
				fmt.Println("runPhp: no err")
				started <- true
				return
			}
		}
	}()

	err = cmd.Start()
	if err != nil {
		return
	}

	// fmt.Println("Waiting for stdout")
	// line := <-stdout
	// fmt.Println(line)

	select {
	case <-timeout:
		fmt.Println("runPhp: killing")
		cmd.Process.Kill()
		err = errors.New("command didn't start")
		return
	case <-started:
		return
	}
}

func makeCmd(args []string) (cmd *exec.Cmd, stdout, stderr chan string) {
	cmd = exec.Command(args[0], args[1:]...)

	_stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stdout = chanify(&_stdout)

	_stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	stderr = chanify(&_stderr)

	return
}

func chanify(pipe *io.ReadCloser) (ch chan string) {
	ch = make(chan string)
	scanner := bufio.NewScanner(*pipe)

	go func() {
		for scanner.Scan() {
			ch <- scanner.Text()
		}
		err := scanner.Err()
		if err != nil {
			panic(err)
		}
	}()

	return
}
