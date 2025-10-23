package cmd

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ncw/directio"
)

// our primary container for the io checks
type Exporter struct {
	conf    *Config
	alloc   *Alloc
	metrics *Metrics
}

type Result struct {
	result  bool
	elapsed float64
}

func NewExporter(conf *Config, alloc *Alloc, metrics *Metrics) *Exporter {
	return &Exporter{
		conf:    conf,
		alloc:   alloc,
		metrics: metrics,
	}
}

// starts the primary go-routine, which will run the io checks for ever
func (exp *Exporter) RunIOchecks() *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for {
			var res_r, res_w Result

			exp.alloc.Clean()

			if exp.conf.WriteMode {
				res_w = exp.measure(O_W)
				slog.Debug("elapsed write time", "elapsed", res_w.elapsed, "result", res_w.result)
			}

			if exp.conf.ReadMode {
				res_r = exp.measure(O_R)
				slog.Debug("elapsed read time", "elapsed", res_r.elapsed, "result", res_r.result)
			}

			if (exp.conf.WriteMode && exp.conf.ReadMode) && (res_r.result && res_w.result) {
				if !exp.alloc.Compare() {
					res_r.result = false
				}
			}

			exp.metrics.Set(res_r, res_w)

			time.Sleep(time.Duration(exp.conf.Sleeptime) * time.Second)
		}
	}()

	return &wg
}

// call an io measurement and collect time needed
func (exp *Exporter) measure(mode int) Result {
	start := time.Now()

	result := exp.runExporter(mode)

	// ns => s
	now := time.Now()
	elapsed := float64(now.Sub(start).Nanoseconds()) / 10000000000

	// makes no sense to measure latency if operation failed
	if !result {
		elapsed = 0
	}

	return Result{elapsed: elapsed, result: result}
}

// Calls runcheck's with context timeout
func (exp *Exporter) runExporter(mode int) bool {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Duration(exp.conf.Timeout)*time.Second)
	defer cancel()

	run := make(chan struct{}, 1)
	var res bool

	go func() {
		switch mode {
		case O_R:
			res = exp.runcheck_r()
		case O_W:
			res = exp.runcheck_w()
		}
		run <- struct{}{}
	}()

	for {
		select {
		case <-ctx.Done():
			return report(ctx.Err(), nil)
		case <-run:
			return res
		}
	}
}

// Checks file io on the specified path:
//
// - opens it for reading
// - reads the block
// - closes file again
//
// Returns false if anything failed during that sequence,
// true otherwise.
func (exp *Exporter) runcheck_r() bool {
	// read
	in, err := directio.OpenFile(exp.conf.File, os.O_RDONLY, 0640)
	if err != nil {
		report(err, nil)
	}

	n, err := io.ReadFull(in, exp.alloc.readBlock)
	if err != nil {
		return report(err, in)
	}

	if n != len(exp.alloc.writeBlock) {
		return report(errors.New("failed to read block"), in)
	}

	if err := in.Close(); err != nil {
		return report(err, nil)
	}

	return true
}

// Checks file io on the specified path:
//
// - open the file (create if it doesnt exist)
// - truncate it if it already exists
// - write some data to it
// - closes the file
//
// Returns false if anything failed during that sequence,
// true otherwise.
func (exp *Exporter) runcheck_w() bool {
	// write
	fd, err := directio.OpenFile(exp.conf.File, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0640)
	if err != nil {
		report(err, nil)
	}

	for i := 0; i < len(exp.alloc.writeBlock); i++ {
		exp.alloc.writeBlock[i] = 'A'
	}

	n, err := fd.Write(exp.alloc.writeBlock)
	if err != nil {
		return report(err, fd)
	}

	if n != len(exp.alloc.writeBlock) {
		return report(errors.New("failed to write block"), fd)
	}

	if err := fd.Close(); err != nil {
		return report(err, nil)
	}

	return true
}
