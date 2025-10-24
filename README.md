[![Actions](https://github.com/tlinden/io-exporter/actions/workflows/ci.yaml/badge.svg)](https://github.com/tlinden/io-exporter/actions)
[![License](https://img.shields.io/badge/license-GPL-blue.svg)](https://github.com/tlinden/io-exporter/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/tlinden/io-exporter)](https://goreportcard.com/report/github.com/tlinden/io-exporter)

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
-t --timeout   <int>          When should the operation timeout in seconds
-s --sleeptime <int>          Time to sleep between checks (default: 5s)
-l --label     <label=value>  Add label to exported metric
-i --internals                Also add labels about resource usage
-r --read                     Only execute the read test
-w --write                    Only execute the write test
-d --debug                    Enable debug log level
-h --help                     Show help
-v --version                  Show program version
```

## Output

Given this command:

```default
io-exporter -l foo=bar -l blah=blubb t/blah
```

You'll get such metrics:

```default
# HELP io_exporter_io_operation whether io is working on the pvc, 1=ok, 0=fail
# TYPE io_exporter_io_operation gauge
io_exporter_io_operation{blah="blubb",exectime="1761148383705",file="t/blah",foo="bar",maxwait="1"} 1
# HELP io_exporter_io_read_latency how long does the read operation take in seconds
# TYPE io_exporter_io_read_latency gauge
io_exporter_io_read_latency{blah="blubb",exectime="1761148383705",file="t/blah",foo="bar",maxwait="1"} 0.0040411716
# HELP io_exporter_io_write_latency how long does the write operation take in seconds
# TYPE io_exporter_io_write_latency gauge
io_exporter_io_write_latency{blah="blubb",exectime="1761148383705",file="t/blah",foo="bar",maxwait="1"} 0
```

You may  also restrict the exporter  to only test read  (`-r` flag) or
write (`-w` flag) operation.

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

## Grafana

I provide a [sample dashboard](grafana), which you can add to your grafana or use
as a starting point to integrate it into your monitoring setup.

It looks like this:

![Screenshot](https://github.com/TLINDEN/io-exporter/blob/main/grafana/screenshot.png)

# Report bugs

[Please open an issue](https://github.com/TLINDEN/io-exporter/issues). Thanks!

# License

This work is licensed under the terms of the General Public Licens
version 3.

# Author

Copyleft (c) 2025 Thomas von Dein
