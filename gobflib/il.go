//go:generate ./genconsts.sh
//go:generate stringer -type=ILBlockType

package gobflib

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"text/template"
)

const (
	DefaultDataSize = 100000
)

const FileGoBody = "templates/body.go"

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

func (b *ILBlock) Dump(out io.Writer, indent int) {
	const indentWidth = 4
	fmt.Fprintf(out, "%*s----------\n", indent*indentWidth, "")
	if b == nil {
		fmt.Fprintf(out, "%*s<nil>\n", indent*indentWidth, "")
		return
	}
	fmt.Fprintf(out, "%*s|Type: %v\n", indent*indentWidth, "", b.typ)
	fmt.Fprintf(out, "%*s|Param:%d\n", indent*indentWidth, "", b.param)
	fmt.Fprintf(out, "%*s|Inner:\n", indent*indentWidth, "")
	if b.inner == nil {
		fmt.Fprintf(out, "%*s<nil>\n", (indent+1)*indentWidth, "")
		return
	}
	for _, ib := range b.inner {
		ib.Dump(out, indent+1)
	}
}

// Equal recursively checks if the implicit ILBlock is identical to
// ILBlock a.
func (b *ILBlock) Equal(a *ILBlock) bool {
	if (b == nil && a != nil) || (b != nil && a == nil) {
		return false
	}
	if b.typ != a.typ {
		return false
	}
	// requiring param to always be equal (even for ILList) is pretty strict
	if b.param != a.param {
		return false
	}
	if len(b.inner) != len(a.inner) {
		return false
	}
	for i := range b.inner {
		if !b.inner[i].Equal(a.inner[i]) {
			return false
		}
	}
	return true
}

func (b *ILBlock) Optimize() {
	// base condition
	if b.typ != ILList && b.typ != ILLoop {
		return
	}

	var wg sync.WaitGroup

	// rip through inner ILBlocks
	oldinner := b.inner
	b.inner = make([]*ILBlock, 0)

	var lastb *ILBlock
	for _, ib := range oldinner {
		if ib.typ == ILList || ib.typ == ILLoop {
			b.Append(ib)
			wg.Add(1)
			go func(wg *sync.WaitGroup, ib *ILBlock) {
				ib.Optimize()
				wg.Done()
			}(&wg, ib)
			lastb = nil
		} else {
			/* Combine DataAdds, DataPtrAdds, and WriteBs */
			if lastb != nil && lastb.typ == ib.typ {
				lastb.param += ib.param
			} else {
				b.Append(ib)
				lastb = ib
			}
		}
	}
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
		out.WriteString("for data[datap] != 0 {\n")
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
		out.WriteString(fmt.Sprintf("writeb(%v)\n", b.param))
	}
	return out.String()
}

type TemplateBody struct {
	InitialDataSize int
	Body            *ILBlock
}

func (p *BFProgram) CreateILTree() *ILBlock {
	s := NewILBlockStack()
	ib := NewILBlock(ILList)
	var cur = ib
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
	return ib
}

func (p *BFProgram) GenGo(output io.Writer) error {
	var usegofmt bool = true
	var err error
	_, err = exec.LookPath("gofmt")
	if err != nil {
		usegofmt = false
	}

	var body TemplateBody
	body.InitialDataSize = DefaultDataSize
	body.Body = p.CreateILTree()
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
