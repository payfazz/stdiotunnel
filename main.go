package main

import (
	"fmt"
	"os"

	"github.com/payfazz/go-errors-ext/errhandlerext"
	"github.com/payfazz/go-errors/errhandler"
	"github.com/payfazz/stdlog"
)

func main() {
	defer errhandler.With(errhandlerext.LogAndExit)

	if len(os.Args) != 3 {
		showUsage()
	}

	switch os.Args[1] {
	case "server":
		runServer(os.Args[2])
	case "client":
		runClient(os.Args[2])
	default:
		showUsage()
	}
}

func showUsage() {
	stdlog.Err.Print(fmt.Sprintf(
		"Usage:\n"+
			"%s client <network:addr>\n"+
			"%s server <network:addr>\n"+
			"\n"+
			"network and addr are as described in https://golang.org/pkg/net/\n"+
			"\n"+
			"Example:\n"+
			"%s client tcp:127.0.0.1:8080\n"+
			"%s server tcp::8080\n",
		os.Args[0], os.Args[0], os.Args[0], os.Args[0],
	))
	os.Exit(1)
}
