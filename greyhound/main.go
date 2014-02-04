package main

import (
	"flag"
	"fmt"
	"github.com/holizz/greyhound"
	"log"
	"net/http"
	"time"
)

func main() {
	port := flag.Int("p", 3000, "port number to listen on")
	dir := flag.String("d", ".", "directory to serve")
	timeout := flag.Duration("t", time.Second * 5, "timeout in milliseconds")
	flag.Parse()

	phpHandler, err := greyhound.NewPhpHandler(*dir, *timeout, []string{})
	if err != nil {
		log.Fatalln(err)
	}

	http.Handle("/", phpHandler)

	fmt.Printf("Listening on :%d\n", *port)
	log.Fatalln(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
