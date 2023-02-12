package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pin/tftp/v3"
)

// Only used for testing

func main() {
	host := flag.String("host", "localhost", "host to connect to")
	port := flag.Int("port", 69, "port to connect to")
	filename := flag.String("filename", "", "filename to download")
	outputDir := flag.String("outputdir", "output", "output directory")
	flag.Parse()
	if *filename == "" {
		log.Fatal("filename is required!")
	}
	c, err := tftp.NewClient(fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		log.Fatal(err)
	}
	wt, err := c.Receive(*filename, "octet")
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Create(fmt.Sprintf("%s%c%s", *outputDir, os.PathSeparator, *filename))
	if err != nil {
		log.Fatal(err)
	}
	// Optionally obtain transfer size before actual data.
	if n, ok := wt.(tftp.IncomingTransfer).Size(); ok {
		fmt.Printf("Transfer size: %d\n", n)
	}
	n, err := wt.WriteTo(file)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d bytes received\n", n)
}
