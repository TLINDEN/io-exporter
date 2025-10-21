package cmd

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// custom labels
type Label struct {
	Name, Value string
}

// simple prometheus wrapper
type Metrics struct {
	run      *prometheus.GaugeVec
	latency  *prometheus.GaugeVec
	registry *prometheus.Registry
	values   []string
}

func NewMetrics(conf *Config) *Metrics {
	labels := []string{"file", "maxwait"}
	LabelLen := 2

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
		latency: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "io_exporter_io_latency",
				Help: "how long does the operation take in seconds",
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
			metrics.latency,

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
		metrics.registry.MustRegister(metrics.run, metrics.latency)
	}

	// static labels
	metrics.values[0] = conf.File
	metrics.values[1] = fmt.Sprintf("%d", conf.Timeout)

	// custom labels via -l label=value
	for idx, label := range conf.Labels {
		metrics.values[idx+LabelLen] = label.Value
	}

	return metrics
}

func (metrics *Metrics) Set(result bool, elapsed float64) {
	var res float64

	if result {
		res = 1
	}

	metrics.run.WithLabelValues(metrics.values...).Set(res)
	metrics.latency.WithLabelValues(metrics.values...).Set(elapsed)
}
