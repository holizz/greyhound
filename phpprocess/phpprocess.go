package phpprocess

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"time"
)

type PhpProcess struct {
	dir string
	port int
	host string
	cmd *exec.Cmd
}

func NewPhpProcess(dir string) (ph *PhpProcess, err error) {
	for p := 8001; p < int(math.Pow(2, 16)); p++ {
		ph = &PhpProcess{
			dir: dir,
			// Use 127.0.0.1 here instead of localhost
			// otherwise PHP only listens on ::1
			host: fmt.Sprintf("127.0.0.1:%d", p),
		}
		ph.cmd, err = runPhp(ph.dir, ph.host)
		if err == nil {
			return
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
	u := r.URL
	u.Scheme = "http"
	u.Host = ph.host

	resp, err := http.Get(u.String())
	if err != nil {
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	bufWriter := bufio.NewWriter(w)
	bufWriter.ReadFrom(resp.Body)
	bufWriter.Flush()

	return
}

func runPhp(dir string, host string) (cmd *exec.Cmd, err error) {
	cmd = exec.Command(
		"php",
		"-n", // do not read php.ini
		"-S", host,
		"-t", dir,
		"-d", "display_errors=Off",
		"-d", "log_errors=On",
		"-d", "error_reporting=E_ALL",
	)

	err = cmd.Start()

	if err != nil {
		return
	}

	e := make(chan error)

	go func() {
		e <- cmd.Wait()
	}()

	select {
	case <-time.After(time.Second):
		return
	case err = <-e:
		return
	}
}
