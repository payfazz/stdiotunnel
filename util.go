package main

import (
	"io"
	"net/http"
)

type copyAllParam struct {
	terminateCh chan struct{}
	reader      io.Reader
	writer      io.Writer
}

func copyAll(p copyAllParam) error {
	w := p.writer
	r := p.reader
	f, fOk := w.(http.Flusher)
	buf := [1 << 20]byte{}
	for {
		select {
		case <-p.terminateCh:
			return nil
		default:
			nr, er := r.Read(buf[:])
			if nr > 0 {
				nw, ew := w.Write(buf[0:nr])
				if fOk {
					f.Flush()
				}
				if ew != nil {
					return ew
				}
				if nr != nw {
					return io.ErrShortWrite
				}
			}
			if er != nil {
				if er == io.EOF {
					return nil
				}
				return er
			}
		}
	}
}

type eofnotifier struct {
	backend io.Reader
	ch      chan struct{}
}

func (en eofnotifier) Read(data []byte) (int, error) {
	n, err := en.backend.Read(data)
	if err == io.EOF {
		select {
		case <-en.ch:
		default:
			close(en.ch)
		}
	}
	return n, err
}
