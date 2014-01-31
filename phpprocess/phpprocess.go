package phpprocess

import (
	"errors"
	// "net/http"
	"os"
	"time"
)

// type PhpProcess struct {
// 	dir string
// 	port int
// 	host string
// 	cmd *exec.Cmd
// }

// func NewPhpProcess(dir string) (ph *PhpProcess, err error) {
// 	ph = &PhpProcess{
// 		dir: dir,
// 		host: "localhost:8001",
// 	}
// 	ph.cmd, err = runPhp(ph.dir, ph.host)
// 	return
// }

// func (ph PhpProcess)Close() {
// }

// func (ph PhpProcess)MakeRequest(w http.ResponseWriter, r *http.Request) {
// }

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
