package main

import (
	"flag"
	"fmt"
	"github.com/codegangsta/martini"
	"github.com/holizz/greyhound"
	"log"
	"net/http"
	"os"
	"time"
)

type stringslice []string

func (s *stringslice) String() string {
	return "a"
}

func (s *stringslice) Set(in string) (err error) {
	*s = append(*s, in)
	return
}

func main() {
	ignore := stringslice([]string{})
	port := flag.Int("p", 3000, "port number to listen on")
	dir := flag.String("d", ".", "directory to serve")
	timeout := flag.Duration("t", time.Second * 5, "timeout in milliseconds")
	flag.Var(&ignore, "i", "ignore errors matching this string (multiple allowed)")

	flag.Usage = func () {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] -- [php options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	queue := greyhound.NewQueuedPhpHandler(5, *dir, *timeout, flag.Args(), ignore)

	fallbackHandler := greyhound.NewFallbackHandler(*dir, ".php")

	m := martini.Classic()
	m.Handlers(
		fallbackHandler.ServeHTTP,
		queue.ServeHTTP,
	)

	http.Handle("/", m)

	fmt.Printf("Listening on :%d\n", *port)
	log.Fatalln(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
