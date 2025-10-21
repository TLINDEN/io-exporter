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

	metrics := NewMetrics(conf)
	alloc := NewAlloc()

	setLogger(os.Stdout)

	go func() {
		for {
			start := time.Now()

			result := runExporter(conf.File, alloc, time.Duration(conf.Timeout)*time.Second)

			// ns => s
			now := time.Now()
			elapsed := float64(now.Sub(start).Nanoseconds()) / 10000000000

			metrics.Set(result, elapsed)

			time.Sleep(SLEEP * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(
		metrics.registry,
		promhttp.HandlerOpts{},
	))

	slog.Info("start testing and serving metrics on localhost", "port", conf.Port)
	slog.Info("test setup", "file", conf.File, "labels", strings.Join(conf.Label, ","))
	http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), nil)
}
