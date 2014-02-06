package main

import (
	"flag"
	"fmt"
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
	flag.Parse()

	if len(flag.Args()) > 0 {
		flag.Usage()
		os.Exit(0)
	}

	phpHandler, err := greyhound.NewPhpHandler(*dir, *timeout, ignore)
	defer phpHandler.Close()
	if err != nil {
		log.Fatalln(err)
	}

	fallbackHandler := greyhound.NewFallbackHandler(*dir, ".php", phpHandler)

	http.Handle("/", fallbackHandler)

	fmt.Printf("Listening on :%d\n", *port)
	log.Fatalln(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
