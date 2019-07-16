package main

import (
	"io"
	"net/http"

	"github.com/payfazz/go-errors"
)

func copyAll(r io.Reader, w io.Writer) error {
	f, flushable := w.(http.Flusher)
	buf := [1 << 20]byte{}
	for {
		nr, rErr := r.Read(buf[:])
		if nr > 0 {
			nw, wErr := w.Write(buf[0:nr])
			if flushable {
				f.Flush()
			}
			if wErr != nil {
				return errors.Wrap(wErr)
			}
			if nr != nw {
				return errors.Wrap(io.ErrShortWrite)
			}
		}
		if rErr != nil {
			if rErr == io.EOF {
				return nil
			}
			return errors.Wrap(rErr)
		}
	}
}

type eofNotifier struct {
	backend io.Reader
	closeCh chan struct{}
}

func newEOFNotifier(b io.Reader) *eofNotifier {
	return &eofNotifier{
		backend: b,
		closeCh: make(chan struct{}),
	}
}

func (e *eofNotifier) ch() <-chan struct{} {
	return e.closeCh
}

func (e *eofNotifier) Read(data []byte) (int, error) {
	n, err := e.backend.Read(data)
	if err == io.EOF {
		select {
		case <-e.closeCh:
		default:
			close(e.closeCh)
		}
	}
	return n, err
}
