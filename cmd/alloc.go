package cmd

import "github.com/ncw/directio"

// aligned allocs used for testing
type Alloc struct {
	writeBlock []byte
	readBlock  []byte
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
