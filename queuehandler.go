package greyhound

import (
	"net/http"
	"time"
)

type QueuedPhpHandler struct {
	Pool    chan *PhpHandler
	Dir     string
	Timeout time.Duration
	Args    []string
	Ignore  []string
	die     chan bool
}

func NewQueuedPhpHandler(poolSize int, dir string, timeout time.Duration, args, ignore []string) *QueuedPhpHandler {
	qh := &QueuedPhpHandler{
		Pool:    make(chan *PhpHandler, poolSize),
		Dir:     dir,
		Timeout: timeout,
		Args:    args,
		Ignore:  ignore,
		die:     make(chan bool),
	}

	go qh.createPhpHandlers()

	return qh
}

func (qh *QueuedPhpHandler) createPhpHandlers() {
	for {
		ph, err := NewPhpHandler(qh.Dir, qh.Timeout, qh.Args, qh.Ignore)
		if err != nil {
			panic(err)
		}
		qh.Pool <- ph
	}
}

func (qh *QueuedPhpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ph := <-qh.Pool
	defer ph.Close()
	ph.ServeHTTP(w, r)
}

func (qh *QueuedPhpHandler) Close() {
	qh.die <- true
	for {
		ph, ok := <-qh.Pool
		if !ok {
			break
		}
		defer ph.Close()
	}
}
