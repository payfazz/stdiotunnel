package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/payfazz/go-errors"
	"github.com/payfazz/go-errors/errhandler"
	"github.com/payfazz/ioconn"
	"github.com/payfazz/stdlog"
	"golang.org/x/net/http2"
)

func runServer(addr string) {
	leftAddr := strings.SplitN(addr, ":", 2)
	if len(leftAddr) != 2 {
		showUsage()
	}

	leftListener, err := net.Listen(leftAddr[0], leftAddr[1])
	errhandler.Check(errors.Wrap(err))

	stdlog.Err.Print(fmt.Sprintf("Server listening on %v\n", leftListener.Addr()))

	wrappedStdin := newEOFNotifier(os.Stdin)
	stdinClosed := wrappedStdin.ch()

	rightMuxed := ioconn.New(ioconn.Config{
		Reader: wrappedStdin,
		Writer: os.Stdout,
	})

	go func() {
		<-stdinClosed
		leftListener.Close()
	}()

	rightMuxedConn, err := (&http2.Transport{}).NewClientConn(rightMuxed)
	errhandler.Check(errors.Wrap(err))
	defer rightMuxedConn.Close()

mainloop:
	for {
		left, err := leftListener.Accept()
		if err != nil {
			select {
			case <-stdinClosed:
				break mainloop
			default:
				errors.PrintTo(stdlog.Err, errors.Wrap(err))
				time.Sleep(100 * time.Millisecond)
				continue mainloop
			}
		}

		go func() {
			defer errhandler.With(func(err error) {
				errors.PrintTo(stdlog.Err, errors.Wrap(err))
			})
			defer left.Close()

			rightReq := &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "http",
					Host:   "stdiotunnel",
				},
				Body: left,
			}
			rightResp, err := rightMuxedConn.RoundTrip(rightReq)
			errhandler.Check(errors.Wrap(err))
			defer rightResp.Body.Close()

			if err := copyAll(rightResp.Body, left); err != nil {
				select {
				case <-stdinClosed:
				default:
					errors.PrintTo(stdlog.Err, errors.Wrap(err))
				}
			}
		}()
	}
}
