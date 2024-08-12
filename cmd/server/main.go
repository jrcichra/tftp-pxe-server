package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/jrcichra/tftp-pxe-server/internal/server"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	directory := flag.String("directory", ".", "directory to serve")
	port := flag.Int("port", 69, "tftp port")
	metricsPort := flag.Int("metrics-port", 9101, "metrics port")
	timeout := flag.Int("timeout", 60, "seconds for tftp timeouts")
	singlePort := flag.Bool("single-port", false, "run in single port mode")
	defaultFolder := flag.String("default-folder", "default", "folder to serve when IP address isn't overridden")
	flag.Parse()

	var g run.Group

	// wait for signal
	{
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()
		g.Add(func() error {
			<-ctx.Done()
			cancel()
			return ctx.Err()
		}, func(err error) {
			log.Println(err)
		})
	}

	// server
	{
		s := server.Server{
			Directory:     *directory,
			Port:          *port,
			Timeout:       time.Duration(*timeout) * time.Second,
			SinglePort:    *singlePort,
			DefaultFolder: *defaultFolder,
		}
		prometheus.MustRegister(&s)
		g.Add(func() error {
			return s.Run()
		}, func(err error) {
			s.Stop()
		})
	}

	// metrics
	{
		http.Handle("/metrics", promhttp.Handler())
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *metricsPort))
		if err != nil {
			panic(err)
		}
		g.Add(func() error {
			return http.Serve(ln, nil)
		}, func(err error) {
			ln.Close()
		})
	}

	if err := g.Run(); err != nil {
		log.Println(err)
	}

}
