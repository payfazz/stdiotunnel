package main

import (
	"io"
	"os"

	"github.com/payfazz/go-errors"
	"github.com/payfazz/stdlog"
)

type stdioWrapper struct {
	io.Writer
	io.Reader
	readerDoneCh chan struct{}
}

func newStdioWrapper() *stdioWrapper {
	return &stdioWrapper{
		Writer:       os.Stdout,
		Reader:       os.Stdin,
		readerDoneCh: make(chan struct{}),
	}
}

func (w *stdioWrapper) Read(data []byte) (int, error) {
	n, err := w.Reader.Read(data)
	if err == io.EOF {
		select {
		case <-w.readerDoneCh:
		default:
			close(w.readerDoneCh)
		}
	}
	return n, err
}

func (w *stdioWrapper) readerDone() bool {
	select {
	case <-w.readerDoneCh:
		return true
	default:
		return false
	}
}

func (w *stdioWrapper) Close() error { return nil }

func biCopy(left, right io.ReadWriter, globalDone <-chan struct{}) {
	done := make(chan struct{})
	defer close(done)

	leftToRightDone := make(chan struct{})
	rightToLeftDone := make(chan struct{})

	go func() {
		defer close(leftToRightDone)
		if _, err := io.Copy(left, right); err != nil {
			select {
			case <-globalDone:
			case <-done:
			default:
				errors.PrintTo(stdlog.Err, errors.Wrap(err))
			}
		}
	}()

	go func() {
		defer close(rightToLeftDone)
		if _, err := io.Copy(right, left); err != nil {
			select {
			case <-globalDone:
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
}
