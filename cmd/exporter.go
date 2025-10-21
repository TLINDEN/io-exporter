package cmd

import (
	"bytes"
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

// Calls runcheck() with timeout
func runExporter(file string, alloc *Alloc, timeout time.Duration) bool {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	run := make(chan struct{}, 1)
	var res bool

	go func() {
		res = runcheck(file, alloc)
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
func runcheck(file string, alloc *Alloc) bool {
	alloc.Clean()

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

	// read
	in, err := directio.OpenFile(file, os.O_RDONLY, 0640)
	if err != nil {
		die(err, nil)
	}

	n, err = io.ReadFull(in, alloc.readBlock)
	if err != nil {
		return die(err, in)
	}

	if n != len(alloc.writeBlock) {
		return die(errors.New("failed to read block"), fd)
	}

	if err := in.Close(); err != nil {
		return die(err, nil)
	}

	// compare
	if !bytes.Equal(alloc.writeBlock, alloc.readBlock) {
		return die(errors.New("read not the same as written"), nil)
	}

	return true
}
