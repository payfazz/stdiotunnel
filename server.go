package main

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/payfazz/ioconn"
	"golang.org/x/net/http2"
)

func runServer(addr string) {
	logger := log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
	logger.Println("Starting server")

	netaddr := strings.SplitN(addr, ":", 2)
	if len(netaddr) != 2 {
		showUsage()
	}

	leftListener, err := net.Listen(netaddr[0], netaddr[1])
	if err != nil {
		logger.Fatalln(err)
		return
	}
	defer leftListener.Close()
	logger.Printf("listening on %v\n", leftListener.Addr())

	rightMuxed := ioconn.New(ioconn.Config{
		Reader: os.Stdin,
		Writer: os.Stdout,
	})

	rightMuxedConn, err := (&http2.Transport{}).NewClientConn(rightMuxed)
	if err != nil {
		logger.Println(err)
		return
	}

	for {
		left, err := leftListener.Accept()
		if err != nil {
			logger.Println(err)
			continue
		}

		go func() {
			defer left.Close()

			allCh := make(chan struct{})
			defer close(allCh)

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

			rightToLeftCh := make(chan struct{})
			go func() {
				if err := copyAll(copyAllParam{
					terminateCh: allCh,
					doneCh:      rightToLeftCh,
					reader:      rightResp.Body,
					writer:      left,
				}); err != nil {
					logger.Println(err)
				}
			}()

			<-rightToLeftCh
		}()
	}
}
