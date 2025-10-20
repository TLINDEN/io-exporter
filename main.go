package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	flag "github.com/spf13/pflag"

	"github.com/knadh/koanf/providers/posflag"
	koanf "github.com/knadh/koanf/v2"
	"github.com/ncw/directio"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	Version = `v0.0.1`
	SLEEP   = 5
	Usage   = `io-exporter [options] <file>
Options:
-t --timeout <int>          When should the operation timeout in seconds
-l --label   <label=value>  Add label to exported metric
-h --help                   Show help
-v --version                Show program version`
)

var (
	labels = []string{"file", "maxwait"}
)

type Label struct {
	Name, Value string
}

type Config struct {
	Showversion bool     `koanf:"version"` // -v
	Showhelp    bool     `koanf:"help"`    // -h
	Label       []string `koanf:"label"`   // -v
	Timeout     int      `koanf:"timeout"` // -t
	Port        int      `koanf:"port"`    // -p

	File   string
	Labels []Label
}

func InitConfig(output io.Writer) (*Config, error) {
	var kloader = koanf.New(".")

	// setup custom usage
	flagset := flag.NewFlagSet("config", flag.ContinueOnError)
	flagset.Usage = func() {
		_, err := fmt.Fprintln(output, Usage)
		if err != nil {
			log.Fatalf("failed to print to output: %s", err)
		}
	}

	// parse commandline flags
	flagset.BoolP("version", "v", false, "show program version")
	flagset.BoolP("help", "h", false, "show help")
	flagset.StringArrayP("label", "l", nil, "additional labels")
	flagset.IntP("timeout", "t", 1, "timeout for file operation in seconds")
	flagset.IntP("port", "p", 9187, "prometheus metrics port to listen to")

	if err := flagset.Parse(os.Args[1:]); err != nil {
		return nil, fmt.Errorf("failed to parse program arguments: %w", err)
	}

	// command line setup
	if err := kloader.Load(posflag.Provider(flagset, ".", kloader), nil); err != nil {
		return nil, fmt.Errorf("error loading flags: %w", err)
	}

	// fetch values
	conf := &Config{}
	if err := kloader.Unmarshal("", &conf); err != nil {
		return nil, fmt.Errorf("error unmarshalling: %w", err)
	}

	// arg is the file under test
	if len(flagset.Args()) > 0 {
		conf.File = flagset.Args()[0]
	} else {
		if !conf.Showversion {
			flagset.Usage()
			os.Exit(1)
		}
	}

	for _, label := range conf.Label {
		if len(label) == 0 {
			continue
		}

		parts := strings.Split(label, "=")
		if len(parts) != 2 {
			return nil, errors.New("invalid label spec: " + label + ", expected label=value")
		}

		conf.Labels = append(conf.Labels, Label{Name: parts[0], Value: parts[1]})
	}

	return conf, nil
}

func die(err error, fd *os.File) bool {
	slog.Debug("failed to check io", "error", err)

	if fd != nil {
		if err := fd.Close(); err != nil {
			slog.Debug("failed to close filehandle", "error", err)
		}
	}

	return false
}

func setLogger(output io.Writer) {
	logLevel := &slog.LevelVar{}
	opts := &tint.Options{
		Level:     logLevel,
		AddSource: false,
	}

	logLevel.Set(slog.LevelDebug)

	handler := tint.NewHandler(output, opts)
	logger := slog.New(handler)

	slog.SetDefault(logger)
}

func main() {
	conf, err := InitConfig(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	promRegistry := prometheus.NewRegistry()

	for _, label := range conf.Labels {
		labels = append(labels, label.Name)
	}

	ioexporterRun := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "io_exporter_io_operation",
			Help: "whether io is working on the pvc, 1=ok, 0=fail",
		},
		labels,
	)

	ioexporterLatency := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "io_exporter_io_latency",
			Help: "how long does the operation take in seconds",
		},
		labels,
	)

	promRegistry.MustRegister(ioexporterRun, ioexporterLatency)

	timeoutstr := fmt.Sprintf("%d", conf.Timeout)

	setLogger(os.Stdout)

	go func() {
		for {
			var res float64
			start := time.Now()

			if check(conf.File, time.Duration(conf.Timeout)*time.Second) {
				res = 1
			} else {
				res = 0
			}

			// ns => s
			now := time.Now()
			elapsed := float64(now.Sub(start).Nanoseconds()) / 10000000000

			values := []string{conf.File, timeoutstr}
			for _, label := range conf.Labels {
				values = append(values, label.Value)
			}

			ioexporterRun.WithLabelValues(values...).Set(res)
			ioexporterLatency.WithLabelValues(values...).Set(elapsed)
			time.Sleep(SLEEP * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(
		promRegistry,
		promhttp.HandlerOpts{},
	))

	slog.Info("start testing and serving metrics on localhost", "port", conf.Port)
	slog.Info("test setup", "file", conf.File, "labels", strings.Join(conf.Label, ","))
	http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), nil)
}

// Calls runcheck() with timeout
func check(file string, timeout time.Duration) bool {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	run := make(chan struct{}, 1)
	var res bool

	go func() {
		res = runcheck(file)
		run <- struct{}{}
	}()

	for {
		select {
		case <-ctx.Done():
			return die(ctx.Err(), nil)
		case <-run:
			return res
		}
	}
}

// Checks file io on the specified path:
//
// - open the file (create if it doesnt exist)
// - truncate it if it already exists
// - write some data to it
// - closes the file
// - re-opens it for reading
// - reads the block
// - compares if written block is equal to read block
// - closes file again
//
// Returns false if anything failed during that sequence,
// true otherwise.
func runcheck(file string) bool {
	// write
	fd, err := directio.OpenFile(file, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0640)
	if err != nil {
		die(err, nil)
	}

	block1 := directio.AlignedBlock(directio.BlockSize)
	for i := 0; i < len(block1); i++ {
		block1[i] = 'A'
	}

	n, err := fd.Write(block1)
	if err != nil {
		return die(err, fd)
	}

	if n != len(block1) {
		return die(errors.New("failed to write block"), fd)
	}

	if err := fd.Close(); err != nil {
		return die(err, nil)
	}

	// read
	in, err := directio.OpenFile(file, os.O_RDONLY, 0640)
	if err != nil {
		die(err, nil)
	}

	block2 := directio.AlignedBlock(directio.BlockSize)

	n, err = io.ReadFull(in, block2)
	if err != nil {
		return die(err, in)
	}

	if n != len(block1) {
		return die(errors.New("failed to read block"), fd)
	}

	if err := in.Close(); err != nil {
		return die(err, nil)
	}

	// compare
	if !bytes.Equal(block1, block2) {
		return die(errors.New("Read not the same as written"), nil)
	}

	return true
}
