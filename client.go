package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/payfazz/ioconn"
	"golang.org/x/net/http2"
)

func runClient(addr string) {
	logger := log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)
	logger.Println("Starting client")

	netaddr := strings.SplitN(addr, ":", 2)
	if len(netaddr) != 2 {
		showUsage()
	}

	leftMuxed := ioconn.New(ioconn.Config{
		Reader: os.Stdin,
		Writer: os.Stdout,
	})

	(&http2.Server{}).ServeConn(leftMuxed, &http2.ServeConnOpts{
		BaseConfig: &http.Server{
			ErrorLog: logger,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				leftWriter := w
				leftReader := r.Body

				right, err := net.Dial(netaddr[0], netaddr[1])
				if err != nil {
					logger.Println(err)
					return
				}
				defer right.Close()

				allCh := make(chan struct{})
				defer close(allCh)

				leftToRightCh := make(chan struct{})
				go func() {
					defer close(leftToRightCh)
					if err := copyAll(copyAllParam{
						terminateCh: allCh,
						reader:      leftReader,
						writer:      right,
					}); err != nil {
						select {
						case <-allCh:
						default:
							logger.Println(err)
						}
					}
				}()

				rightToLeftCh := make(chan struct{})
				go func() {
					defer close(rightToLeftCh)
					if err := copyAll(copyAllParam{
						terminateCh: allCh,
						reader:      right,
						writer:      leftWriter,
					}); err != nil {
						select {
						case <-allCh:
						default:
							logger.Println(err)
						}
					}
				}()

				select {
				case <-leftToRightCh:
				case <-rightToLeftCh:
				}
			}),
		},
	})
}
