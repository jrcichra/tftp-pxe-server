package server

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pin/tftp/v3"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	labels              = []string{"ip", "path", "success"}
	tftpFileRequestDesc = prometheus.NewDesc("tftp_file_request", "How many times a particular file was requested", labels, nil)
)

type entry struct {
	ip      string
	path    string
	success bool
}

type Server struct {
	Directory      string
	Port           int
	Timeout        time.Duration
	SinglePort     bool
	DefaultFolder  string
	server         *tftp.Server
	metricsEntries map[entry]float64
	metricsLock    sync.RWMutex
}

func (s *Server) Collect(ch chan<- prometheus.Metric) {
	s.metricsLock.RLock()
	defer s.metricsLock.RUnlock()
	for entry, count := range s.metricsEntries {
		ch <- prometheus.MustNewConstMetric(tftpFileRequestDesc, prometheus.CounterValue, count, entry.ip, entry.path, strconv.FormatBool(entry.success))
	}
}

func (s *Server) Describe(desc chan<- *prometheus.Desc) {
	desc <- tftpFileRequestDesc
}

var _ prometheus.Collector = &Server{}

func (s *Server) Run() error {
	s.metricsLock = sync.RWMutex{}
	s.metricsEntries = make(map[entry]float64)
	s.server = tftp.NewServer(s.readHandler, nil)
	s.server.SetTimeout(s.Timeout)
	if s.SinglePort {
		s.server.EnableSinglePort()
	}
	log.Printf("Serving TFTP reads on port %d...\n", s.Port)
	return s.server.ListenAndServe(fmt.Sprintf(":%d", s.Port))
}

func (s *Server) Stop() {
	if s.server != nil {
		s.server.Shutdown()
	}
}

func (s *Server) read(filename string, rf io.ReaderFrom) (string, string, error) {
	// clean the filename
	filename = strings.ToValidUTF8(filename, "")
	// get the remote's IP address
	ip := rf.(tftp.OutgoingTransfer).RemoteAddr().IP.String()

	path := filepath.Join(s.Directory, ip)
	// if the ip folder doesn't exist, use the default folder
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Join(s.Directory, s.DefaultFolder)
	}

	// add the filename
	path = filepath.Join(path, filename)

	log.Printf("Opening %s...\n", path)
	file, err := os.Open(path)
	if err != nil {
		log.Printf("%v\n", err)
		return ip, path, err
	}
	n, err := rf.ReadFrom(file)
	if err != nil {
		log.Printf("%v\n", err)
		return ip, path, err
	}
	log.Printf("%d bytes sent for %s\n", n, path)
	return ip, path, nil
}

func (s *Server) readHandler(filename string, rf io.ReaderFrom) error {
	// read, then update metrics
	ip, path, err := s.read(filename, rf)
	s.metricsLock.Lock()
	s.metricsEntries[entry{
		ip:      ip,
		path:    path,
		success: err == nil,
	}] += 1
	s.metricsLock.Unlock()
	return err
}
