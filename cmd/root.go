package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Run() {
	conf, err := InitConfig(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	if conf.Showversion {
		fmt.Printf("This is io-exporter version %s\n", Version)
		os.Exit(0)
	}

	metrics := NewMetrics(conf)
	alloc := NewAlloc()

	setLogger(os.Stdout, conf.Debug)

	go func() {
		for {
			var result_r, result_w bool
			var elapsed_w, elapsed_r float64

			alloc.Clean()

			if conf.WriteMode {
				elapsed_w, result_w = measure(conf.File, alloc, conf.Timeout, O_W)
				slog.Debug("elapsed write time", "elapsed", elapsed_w, "result", result_w)
			}

			if conf.ReadMode {
				elapsed_r, result_r = measure(conf.File, alloc, conf.Timeout, O_R)
				slog.Debug("elapsed read time", "elapsed", elapsed_r, "result", result_r)
			}

			if conf.WriteMode && conf.ReadMode {
				if !alloc.Compare() {
					result_r = false
				}
			}

			metrics.Set(result_r, result_w, elapsed_r, elapsed_w)

			time.Sleep(time.Duration(conf.Sleeptime) * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(
		metrics.registry,
		promhttp.HandlerOpts{},
	))

	slog.Info("start testing and serving metrics on localhost", "port", conf.Port)
	slog.Info("test setup", "file", conf.File, "labels", strings.Join(conf.Label, ","))
	slog.Info("measuring", "read", conf.ReadMode, "write", conf.WriteMode, "timeout(s)", conf.Timeout)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), nil); err != nil {
		log.Fatal(err)
	}
}

func measure(file string, alloc *Alloc, timeout int, mode int) (float64, bool) {
	start := time.Now()

	result := runExporter(file, alloc, time.Duration(timeout)*time.Second, mode)

	// ns => s
	now := time.Now()
	elapsed := float64(now.Sub(start).Nanoseconds()) / 10000000000

	return elapsed, result
}
