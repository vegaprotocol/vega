/*
Command dummyriskmodel serves a dummy risk model on a local socket.

Syntax:

    dummyriskmodel -sockpath /path/to/dummyriskmodel.sock
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

const (
	socketName = "dummyriskmodel.sock"
)

var sockPath string

func init() {
	flag.StringVar(&sockPath, "sockpath", "", "path to the socket file")
}

func main() {
	flag.Parse()
	if len(sockPath) <= 0 {
		log.Println("error: missing sockpath flag")
		os.Exit(1)
	}

	// create a unix domain socket
	ln, err := net.Listen("unix", filepath.Join(sockPath, socketName))
	if err != nil {
		log.Printf("error: cannot open socket, %v", err)
		os.Exit(1)
	}

	s := http.Server{
		Handler: NewRiskModel(),
	}
	// be sure to close once we exit or the socket file will not be destroyed
	defer s.Shutdown(context.Background())
	go func() {
		// listen for http in our unix domain socket
		log.Printf("exiting http server: %v", s.Serve(ln))
	}()
	waitSig()
}

func waitSig() {
	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	log.Printf("waiting for exit signals")

	select {
	case sig := <-gracefulStop:
		log.Printf("caught signal %v", fmt.Sprintf("%+v", sig))
	}
}
