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
	"github.com/payfazz/mainutil"
	"github.com/payfazz/stdlog"
	"golang.org/x/net/http2"
)

func runServer(addr string) {
	netaddr := strings.SplitN(addr, ":", 2)
	if len(netaddr) != 2 {
		showUsage()
	}

	leftListener, err := net.Listen(netaddr[0], netaddr[1])
	errhandler.Check(errors.Wrap(err))

	stdlog.E(fmt.Sprintf("Server listening on %v\n", leftListener.Addr()))

	wrappedStdin := newEOFNotifier(os.Stdin)
	stdinClosedCh := wrappedStdin.ch()

	rightMuxed := ioconn.New(ioconn.Config{
		Reader: wrappedStdin,
		Writer: os.Stdout,
	})

	go func() {
		<-stdinClosedCh
		leftListener.Close()
	}()

	rightMuxedConn, err := (&http2.Transport{}).NewClientConn(rightMuxed)
	errhandler.Check(errors.Wrap(err))

mainloop:
	for {
		left, err := leftListener.Accept()
		if err != nil {
			select {
			case <-stdinClosedCh:
				break mainloop
			default:
				mainutil.Eprint(errors.Wrap(err))
				time.Sleep(1 * time.Second)
				continue mainloop
			}
		}

		go func() {
			defer errhandler.With(mainutil.Eprint)
			defer left.Close()

			rightReq := &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: "http",
					Path:   "/",
				},
				Body: left,
			}
			rightResp, err := rightMuxedConn.RoundTrip(rightReq)
			errhandler.Check(errors.Wrap(err))
			defer rightResp.Body.Close()

			if err := copyAll(rightResp.Body, left); err != nil {
				select {
				case <-stdinClosedCh:
				default:
					mainutil.Eprint(errors.Wrap(err))
				}
			}
		}()
	}
}
