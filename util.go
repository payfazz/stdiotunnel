package main

import (
	"io"
	"net/http"
)

type copyAllParam struct {
	doneCh      chan struct{}
	terminateCh chan struct{}
	reader      io.Reader
	writer      io.Writer
}

func copyAll(p copyAllParam) error {
	defer close(p.doneCh)
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
					select {
					case <-p.terminateCh:
						return nil
					default:
						return ew
					}
				}
				if nr != nw {
					select {
					case <-p.terminateCh:
						return nil
					default:
						return io.ErrShortWrite
					}
				}
			}
			if er != nil {
				if er == io.EOF {
					return nil
				}
				select {
				case <-p.terminateCh:
					return nil
				default:
					return er
				}
			}
		}
	}
}
