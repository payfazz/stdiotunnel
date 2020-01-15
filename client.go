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
	"github.com/payfazz/mainutil"
	"github.com/payfazz/stdlog"
	"golang.org/x/net/http2"
)

func runClient(addr string) {
	stdlog.Err.Print("Starting client\n")

	netaddr := strings.SplitN(addr, ":", 2)
	if len(netaddr) != 2 {
		showUsage()
	}

	wrappedStdin := newEOFNotifier(os.Stdin)
	stdinClosedCh := wrappedStdin.ch()

	leftMuxed := ioconn.New(ioconn.Config{
		Reader: wrappedStdin,
		Writer: os.Stdout,
	})

	server := &http2.Server{}

	server.ServeConn(leftMuxed, &http2.ServeConnOpts{
		BaseConfig: &http.Server{
			ErrorLog: log.New(stdlog.Err, "internal http2 error: ", 0),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				leftWriter := w
				leftReader := r.Body

				right, err := net.Dial(netaddr[0], netaddr[1])
				errhandler.Check(errors.Wrap(err))
				defer right.Close()

				halfDoneCh := make(chan struct{})

				leftToRightCh := make(chan struct{})
				go func() {
					defer close(leftToRightCh)
					if err := copyAll(leftReader, right); err != nil {
						select {
						case <-stdinClosedCh:
						case <-halfDoneCh:
						default:
							errors.PrintTo(mainutil.Err, errors.Wrap(err))
						}
					}
				}()

				rightToLeftCh := make(chan struct{})
				go func() {
					defer close(rightToLeftCh)
					if err := copyAll(right, leftWriter); err != nil {
						select {
						case <-stdinClosedCh:
						case <-halfDoneCh:
						default:
							errors.PrintTo(mainutil.Err, errors.Wrap(err))
						}
					}
				}()

				select {
				case <-leftToRightCh:
					close(halfDoneCh)
				case <-rightToLeftCh:
					close(halfDoneCh)
				}
			}),
		},
	})
}
