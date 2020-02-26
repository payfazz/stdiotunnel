package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/payfazz/go-errors"
	"github.com/payfazz/go-errors/errhandler"
	"github.com/payfazz/ioconn"
	"github.com/payfazz/stdlog"
	"golang.org/x/net/http2"
)

func runClient(addr string) {
	stdlog.Err.Print("Starting client\n")

	rightAddr := strings.SplitN(addr, ":", 2)
	if len(rightAddr) != 2 {
		showUsage()
	}

	wrappedStdin := newEOFNotifier(os.Stdin)
	stdinClosed := wrappedStdin.ch()

	leftMuxed := ioconn.New(ioconn.Config{
		Reader: wrappedStdin,
		Writer: os.Stdout,
	})

	server := &http2.Server{}

	server.ServeConn(leftMuxed, &http2.ServeConnOpts{
		BaseConfig: &http.Server{
			ErrorLog: log.New(stdlog.Err, "internal http2 error: ", 0),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer errhandler.With(func(err error) {
					errors.PrintTo(stdlog.Err, errors.Wrap(err))
				})

				leftWriter := w
				leftReader := r.Body

				right, err := net.Dial(rightAddr[0], rightAddr[1])
				errhandler.Check(errors.Wrap(err))
				defer right.Close()

				done := make(chan struct{})
				defer close(done)

				leftToRightDone := make(chan struct{})
				rightToLeftDone := make(chan struct{})

				go func() {
					defer close(leftToRightDone)
					if err := copyAll(leftReader, right); err != nil {
						select {
						case <-stdinClosed:
						case <-done:
						default:
							errors.PrintTo(stdlog.Err, errors.Wrap(err))
						}
					}
				}()

				go func() {
					defer close(rightToLeftDone)
					if err := copyAll(right, leftWriter); err != nil {
						select {
						case <-stdinClosed:
						case <-done:
						default:
							errors.PrintTo(stdlog.Err, errors.Wrap(err))
						}
					}
				}()

				select {
				case <-leftToRightDone:
				case <-rightToLeftDone:
				}
			}),
		},
	})
}
