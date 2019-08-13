package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"

	"github.com/DeedleFake/p9"
)

func main() {
	network := flag.String("net", "unix", "network to listen on")
	addr := flag.String("addr", "/tmp/ipfs.sock", "address to listen on")
	api := flag.String("api", "http://localhost:5001/api", "address of HTTP API for IPFS")
	flag.Parse()

	fs, err := newFS(strings.TrimSuffix(*api, "/") + "/v0")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	lis, err := net.Listen(*network, *addr)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer lis.Close()

	go func() {
		err = p9.Serve(lis, p9.FSConnHandler(fs, 4096))
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
