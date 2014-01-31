package phpprocess

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"
)

type PhpProcess struct {
	dir string
	port int
	host string
	proc *os.Process
}

func NewPhpProcess(dir string) (ph *PhpProcess, err error) {
	for p := 8001; p < int(math.Pow(2, 16)); p++ {
		ph = &PhpProcess{
			dir: dir,
			host: fmt.Sprintf("localhost:%d", p),
		}
		ph.proc, err = runPhp(ph.dir, ph.host)
		if err == nil {
			return
		}
	}
	return nil, errors.New("No free ports found")
}

func (ph *PhpProcess)Close() {
	err := ph.proc.Kill()
	if err != nil {
		panic(err)
	}
}

func (ph *PhpProcess)MakeRequest(w http.ResponseWriter, r *http.Request) {
}

func runPhp(dir string, host string) (proc *os.Process, err error) {
	proc, err = os.StartProcess(
		"/usr/bin/php",
		[]string{
			"-n", // do not read php.ini
			"-S", host,
			"-t", dir,
			"-d", "display_errors=Off",
			"-d", "log_errors=On",
			"-d", "error_reporting=E_ALL",
		},
		&os.ProcAttr{},
	)

	if err != nil {
		return
	}

	e := make(chan error)

	go func() {
		state, err := proc.Wait()
		if err != nil {
			e <- err
		}
		if !state.Success() {
			e <- errors.New("Process returned a non-zero exit status")
		}
		e <- nil
	}()

	select {
	case <-time.After(time.Second):
		return
	case err = <-e:
		return
	}
}
