package greyhound

import (
	"bufio"
	"errors"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

// A PhpProcess represents a single PHP process running the built-in Web server.
//
// Due to the need to check for errors in the STDERR of the process it only allows one call to MakeRequest() at a time (using sync.Mutex).
type PhpProcess struct {
	dir        string
	port       int
	host       string
	cmd        *exec.Cmd
	stderr     *bufio.Reader
	errorLog   chan string
	requestLog chan string
	mutex      *sync.Mutex
	timeout    int
}

// Start up a new PHP server listening on the first free port (between port 8001 and 2^16).
//
// Usage:
// 	ph, err := NewPhpProcess("/path/to/web/root", 1000)
// 	if err != nil {
// 	        panic(err)
// 	}
// 	defer ph.Close()
//
// timeout is in milliseconds
func NewPhpProcess(dir string, timeout int) (ph *PhpProcess, err error) {
	for p := 8001; p < int(math.Pow(2, 16)); p++ {
		ph = &PhpProcess{
			dir: dir,
			// Use 127.0.0.1 here instead of localhost
			// otherwise PHP only listens on ::1
			host: fmt.Sprintf("127.0.0.1:%d", p),
		}
		cmd, stderr, err := runPhp(ph.dir, ph.host)

		if err == nil {
			ph.timeout = timeout
			ph.cmd = cmd
			ph.stderr = bufio.NewReader(stderr)
			ph.errorLog = make(chan string)
			ph.requestLog = make(chan string)
			ph.mutex = &sync.Mutex{}
			go ph.listenForErrors()
			return ph, nil
		}
	}
	return nil, errors.New("No free ports found")
}

// Don't forget to call this!
func (ph *PhpProcess) Close() {
	err := ph.cmd.Process.Kill()
	if err != nil {
		panic(err)
	}
}

// Make a request. Sends an http.Request to the PHP process, writes what it gets to an http.ResponseWriter.
//
// If an error gets printed to STDERR during the request, it shows the error instead of what PHP returned. If the request takes too long it shows a message saying that the request took too long (see timeout option on NewPhpProcess()).
func (ph *PhpProcess) MakeRequest(w http.ResponseWriter, r *http.Request) (err error) {
	ph.mutex.Lock()
	defer ph.mutex.Unlock()

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
		renderTimeout(w, ph.timeout)
		return
	}
	// End timeout stuff

	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Check for errors
	thereWereErrors := false

	// The request gets printed to STDERR only after the errors
	// So it's a reliable way to confirm that the page was returned

FOR:
	for {
		select {
		case <-ph.requestLog:
			break FOR
		case line := <-ph.errorLog:
			renderError(w, line)
			thereWereErrors = true
		}
	}

	if thereWereErrors {
		return
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
func (ph *PhpProcess) listenForErrors() {
	for {
		line, err := ph.stderr.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				return
			} else {
				panic(err)
			}
		}

		if line[25:37] != "] 127.0.0.1:" {
			ph.errorLog <- line[40:]
		} else {
			ph.requestLog <- line[38:]
		}
	}
}

// Render the error template
func renderError(w http.ResponseWriter, s string) {
	w.WriteHeader(http.StatusInternalServerError)

	tmpl := template.Must(template.New("").Parse(`<pre>{{.}}</pre>`))
	tmpl.Execute(w, s)
}

// Render the timeout template
func renderTimeout(w http.ResponseWriter, timeout int) {
	w.WriteHeader(http.StatusInternalServerError)

	tmpl := template.Must(template.New("").Parse(`<p>Waited for {{.}}ms and PHP didn't respond!</p>`))
	tmpl.Execute(w, timeout)
}

// A low-level command
// Starts PHP running, waits one second, returns an error if PHP stopped during that time
func runPhp(dir string, host string) (cmd *exec.Cmd, stderr io.ReadCloser, err error) {
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
	e := make(chan error)

	go func() {
		err := cmd.Wait()
		if err != nil {
			e <- err
		} else {
			e <- errors.New("Command exited early")
		}
	}()

	select {
	case <-time.After(time.Millisecond * 500):
		return
	case err = <-e:
		return
	}
}
