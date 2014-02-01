# greyhound

A CLI tool for serving PHP projects in a more modern fashion.

## Installation/usage

First make sure you have [Go set up and GOPATH defined, etc](http://golang.org/doc/code.html).

    go get github.com/holizz/greyhound/greyhound
    cd /var/vhosts/mysite
    # Serve current directory on port 3000 with a timeout of 5000ms (these are the defaults)
    greyhound -d . -p 3000 -t 5000

It shows you the errors as if you were using Rails or Sinatra - i.e. it doesn't render the page and just shows you the error. Due to needing to listen on STDERR for errors, PhpHandler can only handle one request at a time so it's probably very slow right now.

## Hacking

Documentation: http://godoc.org/github.com/holizz/greyhound

## License

MIT (see LICENSE.txt)
