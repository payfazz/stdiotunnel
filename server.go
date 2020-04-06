package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/payfazz/go-errors"
	"github.com/payfazz/go-errors/errhandler"
	"github.com/payfazz/stdlog"
)

func runServer(addr string) {
	leftAddr := strings.SplitN(addr, ":", 2)
	if len(leftAddr) != 2 {
		showUsage()
	}

	leftListener, err := net.Listen(leftAddr[0], leftAddr[1])
	errhandler.Check(errors.Wrap(err))

	stdlog.Err.Print(fmt.Sprintf("Server listening on %v\n", leftListener.Addr()))

	rightMuxedIO := newStdioWrapper()

	go func() {
		<-rightMuxedIO.readerDoneCh
		leftListener.Close()
	}()

	rightMuxed, err := yamux.Client(rightMuxedIO, nil)
	errhandler.Check(err)
	defer rightMuxed.Close()

	for {
		left, err := leftListener.Accept()
		if err != nil {
			if rightMuxedIO.readerDone() {
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

			right, err := rightMuxed.OpenStream()
			errhandler.Check(err)
			defer right.Close()

			biCopy(left, right, rightMuxedIO.readerDoneCh)
		}()
	}
}
