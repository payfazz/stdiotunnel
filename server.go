package main

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/payfazz/ioconn"
	"golang.org/x/net/http2"
)

func runServer(addr string) {
	logger := log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)

	netaddr := strings.SplitN(addr, ":", 2)
	if len(netaddr) != 2 {
		showUsage()
	}

	leftListener, err := net.Listen(netaddr[0], netaddr[1])
	if err != nil {
		logger.Fatalln(err)
		return
	}

	logger.Printf("Server listening on %v\n", leftListener.Addr())

	allCh := make(chan struct{})
	rightMuxed := ioconn.New(ioconn.Config{
		Reader: eofnotifier{
			backend: os.Stdin,
			ch:      allCh,
		},
		Writer: os.Stdout,
	})

	go func() {
		<-allCh
		leftListener.Close()
	}()

	rightMuxedConn, err := (&http2.Transport{}).NewClientConn(rightMuxed)
	if err != nil {
		logger.Println(err)
		return
	}

mainloop:
	for {
		left, err := leftListener.Accept()
		if err != nil {
			select {
			case <-allCh:
				break mainloop
			default:
				logger.Println(err)
				time.Sleep(1 * time.Second)
				continue mainloop
			}
		}

		go func() {
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
			if err != nil {
				logger.Println(err)
				return
			}
			defer rightResp.Body.Close()

			if err := copyAll(copyAllParam{
				terminateCh: allCh,
				reader:      rightResp.Body,
				writer:      left,
			}); err != nil {
				select {
				case <-allCh:
				default:
					logger.Println(err)
				}
			}
		}()
	}
}
