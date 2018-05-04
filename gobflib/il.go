//go:generate ./genconsts.sh

package gobflib

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"text/template"
)

const (
	DefaultDataSize = 100000
)

const FileGoBody = "templates/body.go"

type ILBlockStack []*ILBlock

func NewILBlockStack() *ILBlockStack {
	s := new(ILBlockStack)
	*s = make([]*ILBlock, 0, 10)
	return s
}

func (s *ILBlockStack) Push(b *ILBlock) {
	*s = append(*s, b)
}

func (s *ILBlockStack) Pop() *ILBlock {
	if len(*s) == 0 {
		return nil
	}
	b := (*s)[len(*s)-1]
	*s = (*s)[0 : len(*s)-1]
	return b
}

type ILBlockType byte

const (
	ILList ILBlockType = iota
	ILLoop
	ILDataPtrAdd
	ILDataAdd
	ILRead
	ILWrite
)

// ILBlock represents an Intermediate Language Block of instruction(s)
type ILBlock struct {
	typ   ILBlockType
	param int64
	inner []*ILBlock
}

func NewILBlock(typ ILBlockType) *ILBlock {
	b := new(ILBlock)
	b.typ = typ
	if b.typ == ILList || b.typ == ILLoop {
		b.inner = make([]*ILBlock, 0, 0)
	}
	return b
}

func NewILBlockFromBFCmd(cmd BFCmd) *ILBlock {
	b := new(ILBlock)

	switch cmd {
	case BFCmdDataPtrIncrement:
		b.typ = ILDataPtrAdd
		b.param = 1
	case BFCmdDataPtrDecrement:
		b.typ = ILDataPtrAdd
		b.param = -1
	case BFCmdDataIncrement:
		b.typ = ILDataAdd
		b.param = 1
	case BFCmdDataDecrement:
		b.typ = ILDataAdd
		b.param = -1
	case BFCmdOutputByte:
		b.typ = ILWrite
		b.param = 1
	case BFCmdInputByte:
		b.typ = ILRead
		b.param = 1
	case BFCmdLoopStart:
		b.typ = ILLoop
		b.inner = make([]*ILBlock, 0, 0)
	case BFCmdLoopEnd:
		b = nil
	default:
		b = nil
	}

	return b
}

func (b *ILBlock) SetParam(param int64) {
	b.param = param
}

func (b *ILBlock) Append(bs ...*ILBlock) {
	b.inner = append(b.inner, bs...)
}

func (b *ILBlock) GetLast() *ILBlock {
	return b.inner[len(b.inner)-1]
}

func (b *ILBlock) Optimize() {
	// base condition
	if b.typ != ILList && b.typ != ILLoop {
		return
	}

	var wg sync.WaitGroup

	oldinner := b.inner
	b.inner = make([]*ILBlock, 0)

	var lastb *ILBlock
	for _, ib := range oldinner {
		if lastb == nil {
			lastb = ib
			continue
		}
		if lastb.typ != ib.typ {
			b.Append(lastb)
			lastb = ib
		} else {
			lastb.param += ib.param
		}

		if ib.typ == ILList || ib.typ == ILLoop {
			wg.Add(1)
			go func(wg *sync.WaitGroup, ib *ILBlock) {
				ib.Optimize()
				wg.Done()
			}(&wg, ib)
		}
	}
	b.Append(lastb)
	wg.Wait()
}

func (b *ILBlock) String() string {
	var out strings.Builder
	switch b.typ {
	case ILList:
		for i := range b.inner {
			out.WriteString(b.inner[i].String())
		}
	case ILLoop:
		out.WriteString("for ; data[datap] != 0; {\n")
		for i := range b.inner {
			out.WriteString(b.inner[i].String())
		}
		out.WriteString("}\n")
	case ILDataPtrAdd:
		out.WriteString(fmt.Sprintf("datapadd(%d)\n", b.param))
	case ILDataAdd:
		delta := byte(b.param)
		out.WriteString(fmt.Sprintf("dataadd(%v)\n", delta))
	case ILRead:
		for i := int64(0); i < b.param; i++ {
			out.WriteString("readb()\n")
		}
	case ILWrite:
		for i := int64(0); i < b.param; i++ {
			out.WriteString("writeb()\n")
		}
	}
	return out.String()
}

type TemplateBody struct {
	InitialDataSize int
	Body            *ILBlock
}

func (p *BFProgram) CreatILTree() *ILBlock {
	s := NewILBlockStack()
	cur := NewILBlock(ILList)
	for _, c := range p.commands {
		if c == BFCmdLoopEnd {
			cur = s.Pop()
			continue
		}

		b := NewILBlockFromBFCmd(c)
		cur.Append(b)

		if c == BFCmdLoopStart {
			s.Push(cur)
			cur = b
		}
	}
	return cur
}

func (p *BFProgram) GenGo(output io.Writer) error {
	var usegofmt bool = true
	var err error
	_, err = exec.LookPath("gofmt")
	if err != nil {
		usegofmt = false
	}

	var body TemplateBody
	body.InitialDataSize = DefaultDataSizeg
	body.Body = p.CreatILTree()
	body.Body.Optimize()

	t := template.Must(template.New("body").Parse(mainfiletemplate))

tryagain:
	if usegofmt {
		gofmt := exec.Command("gofmt")
		pinput, err := gofmt.StdinPipe()
		if err != nil {
			pinput.Close()
			usegofmt = false
			goto tryagain
		}
		gofmt.Stdout = output
		err = gofmt.Start()
		if err != nil {
			pinput.Close()
			return err
		}
		err = t.Execute(pinput, body)
		if err != nil {
			return err
		}
		err = pinput.Close()
		if err != nil {
			usegofmt = false
			goto tryagain
		}
		err = gofmt.Wait()
		if err != nil {
			usegofmt = false
			goto tryagain
		}
	} else {
		err = t.Execute(output, body)
	}

	return err
}
