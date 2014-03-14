package greyhound

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// A PhpHandler represents a single PHP process running the built-in Web server.
//
// Due to the need to check for errors in the STDERR of the process it only allows one call to ServeHTTP() at a time (using sync.Mutex).
type PhpHandler struct {
	dir        string
	host       string
	cmd        *exec.Cmd
	stdout     chan string
	stderr     chan string
	errorLog   chan string
	requestLog chan string
	errorChan  chan error
	timeout    time.Duration
	args       []string
	ignore     []string
	used       bool
}

// NewPhpHandler starts a new PHP server listening on the first free port (between port 8001 and 2^16).
//
// Usage:
// 	ph, err := NewPhpHandler("/path/to/web/root", time.Second)
// 	if err != nil {
// 	        panic(err)
// 	}
// 	defer ph.Close()
//
// timeout is in milliseconds
func NewPhpHandler(dir string, timeout time.Duration, args, ignore []string) (ph *PhpHandler, err error) {
	ph = &PhpHandler{
		dir:     dir,
		timeout: timeout,
		args:    args,
		ignore:  ignore,
	}

	err = ph.start()

	return
}

func (ph *PhpHandler) start() (err error) {
	for p := 8001; p < int(math.Pow(2, 16)); p++ {
		// Use 127.0.0.1 here instead of localhost
		// otherwise PHP only listens on ::1
		ph.host = fmt.Sprintf("127.0.0.1:%d", p)
		cmd, stdout, stderr, errorChan, err := runPhp(ph.dir, ph.host, ph.args)

		if err == nil {
			ph.cmd = cmd
			ph.stdout = stdout
			ph.stderr = stderr
			ph.errorLog = make(chan string)
			ph.requestLog = make(chan string)
			ph.errorChan = errorChan
			ph.used = false
			go ph.listenForErrors()
			ph.logln("start()ed")
			return nil
		}
	}
	err = errors.New("no free ports found")
	return
}

// Close must be called after a successful call to NewPhpHandler otherwise you may get stray PHP processes floating around.
func (ph *PhpHandler) Close() {
	ph.logln("Close")
	ph.used = true
	ph.cmd.Process.Kill()
	// Ignore the error this returns - it probably means the process has already been killed
}

// ServeHTTP sends an http.Request to the PHP process, writes what it gets to an http.ResponseWriter.
//
// If an error gets printed to STDERR during the request, it shows the error instead of what PHP returned. If the request takes too long it shows a message saying that the request took too long (see timeout option on NewPhpHandler).
func (ph *PhpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer ph.Close()

	if ph.used {
		ph.logln("ServeHTTP was called twice!")
		renderError(w, "programmerError", "ServeHTTP cannot be used twice! Create a new PhpHandler.")
		return
	}

	ph.logln(fmt.Sprintf("ServeHTTP (request for %s)", r.URL.String()))

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
		renderError(w, "requestError", fmt.Sprintf("The request could not be performed for an unknown reason: %#v", err))
		return
	}
	defer resp.Body.Close()

	// The request gets printed to STDERR only after the errors
	// So it's a reliable way to confirm that the page was returned

FOR:
	for {
		select {
		case <-ph.errorChan:
			renderError(w, "earlyExitError", "The PHP command exited before it should have.")
			return
		case <-ph.requestLog:
			break FOR
		case line := <-ph.errorLog:
			ignoreError := false

			if !strings.Contains(line, "PHP Fatal error:  ") {
			IGNORE:
				for _, i := range ph.ignore {
					if strings.Contains(line, i) {
						ignoreError = true
						break IGNORE
					}
				}
			}

			if !ignoreError {
				renderError(w, "interpreterError", line)
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
		line := <-ph.stderr
		if len(line) > 37 && line[25:37] != "] 127.0.0.1:" {
			ph.errorLog <- line[27:]
		} else {
			ph.requestLog <- line
		}
	}
}

func (ph *PhpHandler) logln(s string) {
	log.Println(fmt.Sprintf("[%s] %s", ph.host, s))
}
