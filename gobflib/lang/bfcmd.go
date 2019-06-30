package lang

import (
	"github.com/linux4life798/gobf/gobflib/il"
)

type BFCmd byte

const (
	BFCmdDataPtrIncrement BFCmd = iota
	BFCmdDataPtrDecrement
	BFCmdDataIncrement
	BFCmdDataDecrement
	BFCmdInputByte
	BFCmdOutputByte
	BFCmdLoopStart
	BFCmdLoopEnd
	BFCmdUnknown
)

func NewBFCmd(c rune) (bfc BFCmd) {
	switch c {
	case '>':
		bfc = BFCmdDataPtrIncrement
	case '<':
		bfc = BFCmdDataPtrDecrement
	case '+':
		bfc = BFCmdDataIncrement
	case '-':
		bfc = BFCmdDataDecrement
	case '.':
		bfc = BFCmdOutputByte
	case ',':
		bfc = BFCmdInputByte
	case '[':
		bfc = BFCmdLoopStart
	case ']':
		bfc = BFCmdLoopEnd
	default:
		bfc = BFCmdUnknown
	}
	return
}

func (bfc BFCmd) ToILBlock() *il.ILBlock {
	var b *il.ILBlock

	switch bfc {
	case BFCmdDataPtrIncrement:
		b = il.NewILBlock(il.ILDataPtrAdd)
		b.SetParam(1)
	case BFCmdDataPtrDecrement:
		b = il.NewILBlock(il.ILDataPtrAdd)
		b.SetParam(-1)
	case BFCmdDataIncrement:
		b = il.NewILBlock(il.ILDataAdd)
		b.SetParam(1)
	case BFCmdDataDecrement:
		b = il.NewILBlock(il.ILDataAdd)
		b.SetParam(-1)
	case BFCmdOutputByte:
		b = il.NewILBlock(il.ILWrite)
		b.SetParam(1)
	case BFCmdInputByte:
		b = il.NewILBlock(il.ILRead)
		b.SetParam(1)
	case BFCmdLoopStart:
		b = il.NewILBlock(il.ILLoop)
		// b.inner = make([]*il.ILBlock, 0, 0)
	case BFCmdLoopEnd:
		b = nil
	default:
		b = nil
	}

	return b
}

func (bfc BFCmd) String() (c string) {
	switch bfc {
	case BFCmdDataPtrIncrement:
		c = ">"
	case BFCmdDataPtrDecrement:
		c = "<"
	case BFCmdDataIncrement:
		c = "+"
	case BFCmdDataDecrement:
		c = "-"
	case BFCmdOutputByte:
		c = "."
	case BFCmdInputByte:
		c = ","
	case BFCmdLoopStart:
		c = "["
	case BFCmdLoopEnd:
		c = "]"
	default:
		c = "unknown"
	}
	return
}
