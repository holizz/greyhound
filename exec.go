package greyhound

import (
	"bufio"
	"errors"
	"io"
	"os/exec"
	"time"
)

// A low-level command
// Starts PHP running, waits half a second, returns an error if PHP stopped during that time
func runPhp(dir string, host string) (cmd *exec.Cmd, stdout chan string, stderr chan string, errorChan chan error, err error) {
	cmd = exec.Command(
		"php",
		"-n", // do not read php.ini
		"-S", host,
		"-t", dir,
		"-d", "display_errors=Off",
		"-d", "log_errors=On",
		"-d", "error_reporting=E_ALL",
	)

	// Connect stdout
	_stdout, out := cmd.StdoutPipe()
	if out != nil {
		return
	}
	stdout = chanify(&_stdout)

	// Connect stderr
	_stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}
	stderr = chanify(&_stderr)

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
