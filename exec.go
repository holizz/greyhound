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
func runPhp(dir, host string, extraArgs []string) (cmd *exec.Cmd, stdout chan string, stderr chan string, errorChan chan error, err error) {
	args := []string{
		"-n", // do not read php.ini
		"-S", host,
		"-t", dir,
		"-d", "display_errors=Off",
		"-d", "log_errors=On",
		"-d", "error_reporting=E_ALL",
		"-d", "upload_max_filesize=1024G",
		"-d", "post_max_size=1024G",
	}
	args = append(args, extraArgs...)

	cmd = exec.Command("php", args...)

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
	errorChan = make(chan error)

	go func() {
		err = cmd.Run()
		if err != nil {
			errorChan <- err
		} else {
			errorChan <- errors.New("command exited early")
		}
	}()

	// Wait 1 second for the command to terminate
	// If it exits early, that's bad whatever the exit status

	select {
	case <-time.After(time.Second):
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
