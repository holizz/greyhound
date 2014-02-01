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

type PhpProcess struct {
	dir string
	port int
	host string
	cmd *exec.Cmd
	stderr *bufio.Reader
	phpErrors chan string
	mutex *sync.Mutex
}

func NewPhpProcess(dir string) (ph *PhpProcess, err error) {
	for p := 8001; p < int(math.Pow(2, 16)); p++ {
		ph = &PhpProcess{
			dir: dir,
			// Use 127.0.0.1 here instead of localhost
			// otherwise PHP only listens on ::1
			host: fmt.Sprintf("127.0.0.1:%d", p),
		}
		cmd, stderr, err := runPhp(ph.dir, ph.host)

		if err == nil {
			ph.cmd = cmd
			ph.stderr = bufio.NewReader(stderr)
			ph.phpErrors = make(chan string)
			ph.mutex = &sync.Mutex{}
			go ph.listenForErrors()
			return ph, nil
		}
	}
	return nil, errors.New("No free ports found")
}

func (ph *PhpProcess) Close() {
	err := ph.cmd.Process.Kill()
	if err != nil {
		panic(err)
	}
}

func (ph *PhpProcess) MakeRequest(w http.ResponseWriter, r *http.Request) (err error) {
	ph.mutex.Lock()
	defer ph.mutex.Unlock()

	r.URL.Scheme = "http"
	r.URL.Host = ph.host

	// Make the request
	tr := &http.Transport{}

	resp, err := tr.RoundTrip(r)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Check for errors
	thereWereErrors := false
	// Experimentally, 1ms works, 100mcs doesn't work
	c := time.After(time.Millisecond)

	FOR: for {
		select {
		case <-c:
			break FOR
		case line := <-ph.phpErrors:
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
			ph.phpErrors <- line[40:]
		}
	}
}

func renderError(w http.ResponseWriter, s string) {
	w.WriteHeader(http.StatusInternalServerError)

	tmpl := template.Must(template.New("").Parse(`<pre>{{.}}</pre>`))
	tmpl.Execute(w, s)
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
		// Experimentally, 100ms works and 10ms doesn't
	case <-time.After(time.Millisecond*100):
		return
	case err = <-e:
		return
	}
}
