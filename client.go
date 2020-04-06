package main

import (
	"net"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/payfazz/go-errors"
	"github.com/payfazz/go-errors/errhandler"
	"github.com/payfazz/stdlog"
)

func runClient(addr string) {
	rightAddr := strings.SplitN(addr, ":", 2)
	if len(rightAddr) != 2 {
		showUsage()
	}

	stdlog.Err.Print("Starting client\n")

	leftMuxedIO := newStdioWrapper()

	leftMuxed, err := yamux.Server(leftMuxedIO, nil)
	errhandler.Check(err)
	defer leftMuxed.Close()

	for {
		left, err := leftMuxed.AcceptStream()
		if err != nil {
			if leftMuxedIO.readerDone() {
				break
			}
			errors.PrintTo(stdlog.Err, errors.Wrap(err))
			time.Sleep(100 * time.Millisecond)
			continue
		}

		go func() {
			defer errhandler.With(func(err error) {
				errors.PrintTo(stdlog.Err, errors.Wrap(err))
			})

			defer left.Close()

			right, err := net.Dial(rightAddr[0], rightAddr[1])
			errhandler.Check(errors.Wrap(err))
			defer right.Close()

			biCopy(left, right, leftMuxedIO.readerDoneCh)
		}()
	}
}
