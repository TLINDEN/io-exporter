package cmd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/knadh/koanf/providers/posflag"
	koanf "github.com/knadh/koanf/v2"
)

const (
	Version = `v0.0.1`
	SLEEP   = 5
	Usage   = `io-exporter [options] <file>
Options:
-t --timeout   <int>          When should the operation timeout in seconds
-s --sleeptime <int>          Time to sleep between checks (default: 5s)
-l --label     <label=value>  Add label to exported metric
-i --internals                Also add labels about resource usage
-h --help                     Show help
-v --version                  Show program version`
)

// config via commandline flags
type Config struct {
	Showversion bool     `koanf:"version"`   // -v
	Showhelp    bool     `koanf:"help"`      // -h
	Internals   bool     `koanf:"internals"` // -i
	Label       []string `koanf:"label"`     // -v
	Timeout     int      `koanf:"timeout"`   // -t
	Port        int      `koanf:"port"`      // -p
	Sleeptime   int      `koanf:"sleep"`     // -s

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
	flagset.BoolP("internals", "i", false, "add internal metrics")
	flagset.StringArrayP("label", "l", nil, "additional labels")
	flagset.IntP("timeout", "t", 1, "timeout for file operation in seconds")
	flagset.IntP("port", "p", 9187, "prometheus metrics port to listen to")
	flagset.IntP("sleeptime", "s", 5, "time to sleep between checks (default: 5s)")

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
