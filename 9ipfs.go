package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"

	"github.com/DeedleFake/p9"
	"github.com/DeedleFake/p9/proto"
)

func main() {
	addr := flag.String("addr", "$ipfs", "address to listen on")
	api := flag.String("api", "http://localhost:5001/api", "address of HTTP API for IPFS")
	flag.Parse()

	fs, err := newFS(strings.TrimSuffix(*api, "/") + "/v0")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	if p9.IsNamespaceAddr(*addr) {
		os.MkdirAll(p9.NamespaceDir(), 0700)
	}
	network, address := p9.ParseAddr(*addr)

	lis, err := net.Listen(network, address)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer lis.Close()

	go func() {
		err = proto.Serve(lis, p9.Proto(), p9.FSConnHandler(fs, 4096))
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
