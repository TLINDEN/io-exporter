package cmd

import (
	"bytes"
	"errors"

	"github.com/ncw/directio"
)

const (
	O_R = iota
	O_W
	O_RW
)

// aligned allocs used for testing
type Alloc struct {
	writeBlock []byte
	readBlock  []byte
	mode       int
}

// zero the memory blocks
func (alloc *Alloc) Clean() {
	for i := range alloc.writeBlock {
		alloc.writeBlock[i] = 0
	}

	for i := range alloc.readBlock {
		alloc.readBlock[i] = 0
	}
}

func NewAlloc() *Alloc {
	return &Alloc{
		writeBlock: directio.AlignedBlock(directio.BlockSize),
		readBlock:  directio.AlignedBlock(directio.BlockSize),
	}
}

func (alloc *Alloc) Compare() bool {
	// compare
	if !bytes.Equal(alloc.writeBlock, alloc.readBlock) {
		return report(errors.New("read not the same as written"), nil)
	}

	return true
}
