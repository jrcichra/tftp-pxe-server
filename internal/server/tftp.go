package server

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/pin/tftp"
)

type Server struct {
	Directory string
	Port      int
	Timeout   time.Duration
	IPPaths   bool
	server    *tftp.Server
}

func (s *Server) Run() {
	s.server = tftp.NewServer(s.readHandler, nil)
	s.server.SetTimeout(s.Timeout)
	log.Printf("Serving TFTP reads on port %d...\n", s.Port)
	err := s.server.ListenAndServe(fmt.Sprintf(":%d", s.Port))
	if err != nil {
		log.Printf("server: %v\n", err)
		os.Exit(1)
	}
}

func (s *Server) readHandler(filename string, rf io.ReaderFrom) error {
	var path string
	if s.IPPaths {
		// get the remote's IP address
		ip := rf.(tftp.OutgoingTransfer).RemoteAddr().IP
		path = fmt.Sprintf("%s%c%s%c%s", s.Directory, os.PathSeparator, ip.String(), os.PathSeparator, filename)
	} else {
		path = fmt.Sprintf("%s%c%s", s.Directory, os.PathSeparator, filename)
	}

	log.Printf("Opening %s...\n", path)
	file, err := os.Open(path)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	n, err := rf.ReadFrom(file)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	log.Printf("%d bytes sent for %s\n", n, path)
	return nil
}
