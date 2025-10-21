# io-exporter

Report if a given filesystem is operating properly

## Description

This little exporter checks if a filesystem is working properly by
writing and reading to a specified file using direct/io. It reports
the results via HTTP as prometheus metrics. Additional labels can be
specified via commandline.

## Usage

```default
io-exporter [options] <file>
Options:
-t --timeout <int>          When should the operation timeout in seconds
-l --label   <label=value>  Add label to exported metric
-h --help                   Show help
-v --version                Show program version
```

## Output

Given this command:

```default
io-exporter -l foo=bar -l blah=blubb t/blah
```

You'll get such metrics:

```default
# HELP io_exporter_io_latency how long does the operation take in seconds
# TYPE io_exporter_io_latency gauge
io_exporter_io_latency{file="/tmp/blah",maxwait="1",namespace="debug",pod="foo1"} 0.0001142815
# HELP io_exporter_io_operation whether io is working on the pvc, 1=ok, 0=fail
# TYPE io_exporter_io_operation gauge
io_exporter_io_operation{file="/tmp/blah",maxwait="1",namespace="debug",pod="foo1"} 1
```

## Installation

There are no released binaries yet.

### Installation from source

Check out the repository and execute `go build`, then copy the
compiled binary to your `$PATH`.

Or, if you have GNU Make installed, just execute:

```default
go build
```

## Docker

To build:

```default
docker compose build
```

To run locally:

```default
mkdir t
chmod 1777 t
docker compose run -v ./t:/pvc ioexporter /pvc/testfile
```

Or use the pre-build image:

```default
docker run -u `id -u $USER` -v ./t:/pvc ghcr.io/tlinden/io-exporter:latest /pvc/testfile
```

# Report bugs

[Please open an issue](https://github.com/TLINDEN/io-exporter/issues). Thanks!

# License

This work is licensed under the terms of the General Public Licens
version 3.

# Author

Copyleft (c) 2025 Thomas von Dein
