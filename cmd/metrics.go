package cmd

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// custom labels
type Label struct {
	Name, Value string
}

// simple prometheus wrapper
type Metrics struct {
	run       *prometheus.GaugeVec
	latency_r *prometheus.GaugeVec
	latency_w *prometheus.GaugeVec
	registry  *prometheus.Registry
	values    []string
	mode      int
}

func NewMetrics(conf *Config) *Metrics {
	labels := []string{"file", "maxwait", "exectime"}
	LabelLen := 3

	for _, label := range conf.Labels {
		labels = append(labels, label.Name)
	}

	metrics := &Metrics{
		run: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "io_exporter_io_operation",
				Help: "whether io is working on the pvc, 1=ok, 0=fail",
			},
			labels,
		),
		latency_r: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "io_exporter_io_read_latency",
				Help: "how long does the read operation take in seconds",
			},
			labels,
		),
		latency_w: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "io_exporter_io_write_latency",
				Help: "how long does the write operation take in seconds",
			},
			labels,
		),

		// use fixed size slice to avoid repeated allocs
		values: make([]string, LabelLen+len(conf.Labels)),

		registry: prometheus.NewRegistry(),
	}

	if conf.Internals {
		metrics.registry.MustRegister(
			metrics.run,
			metrics.latency_r,
			metrics.latency_w,

			// we  might need  to take  care of the  exporter in  terms of
			// resources, so also report those internals
			collectors.NewGoCollector(
				collectors.WithGoCollectorMemStatsMetricsDisabled(),
			),
			collectors.NewProcessCollector(
				collectors.ProcessCollectorOpts{},
			),
		)
	} else {
		metrics.registry.MustRegister(metrics.run, metrics.latency_r, metrics.latency_w)
	}

	// static labels
	metrics.values[0] = conf.File
	metrics.values[1] = fmt.Sprintf("%d", conf.Timeout)
	metrics.values[2] = fmt.Sprintf("%d", time.Now().UnixMilli())

	// custom labels via -l label=value
	for idx, label := range conf.Labels {
		metrics.values[idx+LabelLen] = label.Value
	}

	switch {
	case conf.ReadMode && conf.WriteMode:
		metrics.mode = O_RW
	case conf.ReadMode:
		metrics.mode = O_R
	case conf.WriteMode:
		metrics.mode = O_W
	}

	return metrics
}

func (metrics *Metrics) Set(result_r, result_w bool, elapsed_r, elapsed_w float64) {
	var res float64

	switch metrics.mode {
	case O_RW:
		if result_r && result_w {
			res = 1
		}
	case O_R:
		if result_r {
			res = 1
		}
	case O_W:
		if result_w {
			res = 1
		}
	}

	metrics.run.WithLabelValues(metrics.values...).Set(res)
	metrics.latency_r.WithLabelValues(metrics.values...).Set(elapsed_r)
	metrics.latency_w.WithLabelValues(metrics.values...).Set(elapsed_w)
}
