package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Main program. starts 2 goroutines: our exporter and the http server
// for  the  prometheus  metrics.  The  exporter  reports  measurement
// results to prometheus metrics directly
func Run() {
	conf, err := InitConfig(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	if conf.Showversion {
		fmt.Printf("This is io-exporter version %s\n", Version)
		os.Exit(0)
	}

	setLogger(os.Stdout, conf.Debug)

	metrics := NewMetrics(conf)
	alloc := NewAlloc()
	exporter := NewExporter(conf, alloc, metrics)

	exporter.RunIOchecks()

	http.Handle("/metrics", promhttp.HandlerFor(
		metrics.registry,
		promhttp.HandlerOpts{},
	))

	slog.Info("io-exporter starting up", "version", Version)
	slog.Info("   serving metrics", "host", "localhost", "port", conf.Port)
	slog.Info("   test setup", "file", conf.File, "labels", strings.Join(conf.Label, ","))
	slog.Info("   measuring", "read", conf.ReadMode, "write", conf.WriteMode, "timeout(s)", conf.Timeout)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), nil); err != nil {
		log.Fatal(err)
	}
}

func report(err error, fd *os.File) bool {
	slog.Debug("failed to check io", "error", err)

	if fd != nil {
		if err := fd.Close(); err != nil {
			slog.Debug("failed to close filehandle", "error", err)
		}
	}

	return false
}
