package cmd

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/ncw/directio"
)

func die(err error, fd *os.File) bool {
	slog.Debug("failed to check io", "error", err)

	if fd != nil {
		if err := fd.Close(); err != nil {
			slog.Debug("failed to close filehandle", "error", err)
		}
	}

	return false
}

// Calls runcheck* with timeout
func runExporter(file string, alloc *Alloc, timeout time.Duration, op int) bool {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	run := make(chan struct{}, 1)
	var res bool

	go func() {
		switch op {
		case O_R:
			res = runcheck_r(file, alloc)
		case O_W:
			res = runcheck_w(file, alloc)
		}
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
// - opens it for reading
// - reads the block
// - closes file again
//
// Returns false if anything failed during that sequence,
// true otherwise.
func runcheck_r(file string, alloc *Alloc) bool {
	// read
	in, err := directio.OpenFile(file, os.O_RDONLY, 0640)
	if err != nil {
		die(err, nil)
	}

	n, err := io.ReadFull(in, alloc.readBlock)
	if err != nil {
		return die(err, in)
	}

	if n != len(alloc.writeBlock) {
		return die(errors.New("failed to read block"), in)
	}

	if err := in.Close(); err != nil {
		return die(err, nil)
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
func runcheck_w(file string, alloc *Alloc) bool {
	// write
	fd, err := directio.OpenFile(file, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0640)
	if err != nil {
		die(err, nil)
	}

	for i := 0; i < len(alloc.writeBlock); i++ {
		alloc.writeBlock[i] = 'A'
	}

	n, err := fd.Write(alloc.writeBlock)
	if err != nil {
		return die(err, fd)
	}

	if n != len(alloc.writeBlock) {
		return die(errors.New("failed to write block"), fd)
	}

	if err := fd.Close(); err != nil {
		return die(err, nil)
	}

	return true
}
