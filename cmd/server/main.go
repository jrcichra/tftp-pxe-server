package main

import (
	"flag"
	"time"

	"github.com/jrcichra/tftp-pxe-server/internal/server"
)

func main() {
	directory := flag.String("directory", ".", "directory to serve")
	port := flag.Int("port", 69, "tftp port")
	timeout := flag.Int("timeout", 10, "seconds for tftp timeouts")
	ipPaths := flag.Bool("ipPaths", true, "prepend request paths with src IP address")
	flag.Parse()
	s := server.Server{
		Directory: *directory,
		Port:      *port,
		Timeout:   time.Duration(*timeout) * time.Second,
		IPPaths:   *ipPaths,
	}
	s.Run()
}
