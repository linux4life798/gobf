package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
)

var ErrUnknownCommand = errors.New("Error: Unknown command in program execution")
var ErrJumpLocationExceedsCommands = errors.New("Error: Jump location exceeds command locations")
var ErrDataPtr = errors.New("Error: Data pointer moved out of bounds (off the beginning)")
var ErrReadError = errors.New("Error: Received read error during runtime")
var ErrWriteError = errors.New("Error: Received write error during runtime")

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

type BFCmd byte

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

const (
	defaultDataSize      = 100000
	defaultJumpStackSize = 10
	debugEnabled         = false
)

type BFProgram struct {
	cmdptr   uint64
	dataptr  uint64
	commands []BFCmd
	data     []byte
	input    io.Reader
	output   io.Writer

	jumpstack    []uint64
	fwdjump      map[uint64]uint64
	revjump      map[uint64]uint64
	appendcmdptr uint64
}

func (p *BFProgram) jumplen() uint64 {
	return uint64(len(p.jumpstack))
}
func (p *BFProgram) jumppush(cmdptr uint64) {
	p.jumpstack = append(p.jumpstack, cmdptr)
}
func (p *BFProgram) jumppop() uint64 {
	cmdptr := p.jumpstack[len(p.jumpstack)-1]
	p.jumpstack = p.jumpstack[0 : len(p.jumpstack)-1]
	return cmdptr
}

func NewBFProgram(initialcommandssize, initialdatasize uint64) *BFProgram {
	p := NewIOBFProgram(initialcommandssize, initialdatasize, os.Stdin, os.Stdout)
	return p
}

func NewIOBFProgram(initialcommandssize, initialdatasize uint64, input io.Reader, output io.Writer) *BFProgram {
	if initialdatasize == 0 {
		initialdatasize = 1
	}
	p := new(BFProgram)
	p.commands = make([]BFCmd, 0, initialcommandssize)
	p.data = make([]byte, initialdatasize)
	p.jumpstack = make([]uint64, 0, defaultJumpStackSize)
	p.fwdjump = make(map[uint64]uint64)
	p.revjump = make(map[uint64]uint64)
	p.input = input
	p.output = output
	return p
}

func (p *BFProgram) Clone() *BFProgram {
	pnew := new(BFProgram)
	pnew.commands = make([]BFCmd, 0, len(p.commands))
	pnew.commands = append(pnew.commands, p.commands...)
	pnew.data = make([]byte, len(p.data))
	pnew.data = append(pnew.data, p.data...)
	pnew.jumpstack = make([]uint64, 0, len(p.jumpstack))
	pnew.jumpstack = append(pnew.jumpstack, p.jumpstack...)
	pnew.fwdjump = make(map[uint64]uint64)
	for k, v := range p.fwdjump {
		pnew.fwdjump[k] = v
	}
	pnew.revjump = make(map[uint64]uint64)
	for k, v := range p.revjump {
		pnew.revjump[k] = v
	}
	pnew.appendcmdptr = p.appendcmdptr
	pnew.cmdptr = p.cmdptr
	pnew.dataptr = p.dataptr
	pnew.input = p.input
	pnew.output = p.output
	return pnew
}

func (p *BFProgram) AppendCommand(cmd rune) {
	c := NewBFCmd(cmd)
	if c == BFCmdUnknown {
		return
	}
	if c == BFCmdLoopStart {
		p.jumppush(p.appendcmdptr)
	}
	if c == BFCmdLoopEnd {
		if p.jumplen() == 0 {
			panic("Unbalanced [ ]")
		}
		openptr := p.jumppop()
		closedptr := p.appendcmdptr
		p.fwdjump[openptr] = closedptr
		p.revjump[closedptr] = openptr
	}
	p.commands = append(p.commands, c)
	p.appendcmdptr++
}

func (p *BFProgram) AppendCommands(cmds ...rune) {
	for _, c := range cmds {
		p.AppendCommand(c)
	}
}

func (p *BFProgram) ReadCommands(in io.Reader) {
	cmdstream := bufio.NewReader(in)
	var ignoreLine = false
	var sameLine = false
	for {
		line, isPrefix, err := cmdstream.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while reading file: %v\n", err)
			os.Exit(1)
		}

		if sameLine && ignoreLine {
			sameLine = isPrefix
			continue
		}

		if !sameLine {
			ignoreLine = false
		}

		for _, c := range line {
			if c == byte('#') {
				ignoreLine = true
				break
			}
			// this will ignore anything but BF characters
			p.AppendCommand(rune(c))
		}

		sameLine = isPrefix
	}
}

func (p *BFProgram) PrintProgram(outio io.Writer) {
	for _, c := range p.commands {
		fmt.Fprintf(outio, "%v", c)
	}
}

func (p *BFProgram) Reset() {
	p.cmdptr = 0
	p.dataptr = 0
	p.data = make([]byte, len(p.data))
}

func (p *BFProgram) RunStep() (bool, error) {

	if debugEnabled {
		fmt.Fprintf(os.Stderr, "PC: %d\n", p.cmdptr)
	}

	// Proper program termination
	if p.cmdptr == uint64(len(p.commands)) {
		return true, nil
	}

	if p.cmdptr > uint64(len(p.commands)) {
		return false, ErrJumpLocationExceedsCommands
	}

	switch p.commands[p.cmdptr] {
	case BFCmdDataPtrIncrement:
		p.dataptr++
		// expand data array if needed
		if p.dataptr >= uint64(len(p.data)) {
			newdata := make([]byte, len(p.data)*2)
			copy(newdata, p.data)
			p.data = newdata
		}
	case BFCmdDataPtrDecrement:
		if p.dataptr == 0 {
			return false, ErrDataPtr
		}
		p.dataptr--
	case BFCmdDataIncrement:
		p.data[p.dataptr]++
	case BFCmdDataDecrement:
		p.data[p.dataptr]--
	case BFCmdInputByte:
		var b [1]byte
		for {
			n, err := p.input.Read(b[:])
			if err != nil {
				return false, ErrReadError
			}
			if n > 0 {
				break
			}
		}
		p.data[p.dataptr] = b[0]

	case BFCmdOutputByte:
		n, err := p.output.Write(p.data[p.dataptr : p.dataptr+1])
		if err != nil {
			return false, ErrWriteError
		}
		if n != 1 {
			return false, ErrWriteError
		}
	case BFCmdLoopStart:
		if p.data[p.dataptr] == 0 {
			p.cmdptr = p.fwdjump[p.cmdptr]
		}
	case BFCmdLoopEnd:
		if p.data[p.dataptr] != 0 {
			p.cmdptr = p.revjump[p.cmdptr]
		}
	default:
		return false, ErrUnknownCommand
	}

	p.cmdptr++

	return false, nil
}

func (p *BFProgram) Run() error {
	for finished, err := p.RunStep(); !finished; finished, err = p.RunStep() {
		if err != nil {
			return err
		}
	}
	return nil
}
