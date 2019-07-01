//go:generate stringer -type=ILBlockType

package il

import (
	"fmt"
	"io"
	"sync"
)

type ILBlockType byte

const (
	ILList ILBlockType = iota
	ILLoop
	ILDataPtrAdd
	ILDataAdd
	ILRead
	ILWrite
	ILDataAddVector
)

// ILBlock represents an Intermediate Language Block of instruction(s)
type ILBlock struct {
	typ   ILBlockType
	param int64
	inner []*ILBlock
	vec   []byte
}

func NewILBlock(typ ILBlockType) *ILBlock {
	b := new(ILBlock)
	b.typ = typ
	if b.typ == ILList || b.typ == ILLoop {
		b.inner = make([]*ILBlock, 0, 0)
	}
	return b
}

func (b *ILBlock) GetType() ILBlockType {
	return b.typ
}

func (b *ILBlock) GetParam() int64 {
	return b.param
}

func (b *ILBlock) GetVector() []byte {
	var v = make([]byte, len(b.vec))
	copy(v, b.vec)
	return v
}

func (b *ILBlock) SetParam(param int64) {
	b.param = param
}

func (b *ILBlock) Append(bs ...*ILBlock) {
	b.inner = append(b.inner, bs...)
}

func (b *ILBlock) GetInner() []*ILBlock {
	return b.inner
}

func (b *ILBlock) GetLast() *ILBlock {
	return b.inner[len(b.inner)-1]
}

func (b *ILBlock) Dump(out io.Writer, indent int) {
	const indentWidth = 4
	fmt.Fprintf(out, "%*s---------------------------------\n", indent*indentWidth, "")
	if b == nil {
		fmt.Fprintf(out, "%*s<nil>\n", indent*indentWidth, "")
		return
	}
	fmt.Fprintf(out, "%*s| %v | param=%v vec=%v |\n", indent*indentWidth, "", b.typ, b.param, b.vec)
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
		} else if ib.typ == ILDataAddVector {
			b.Append(ib)
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

// isPruneable uses a set of rules to determine id an ILBlock
// node is able to be removed.
func (b *ILBlock) isPruneable() bool {
	if b == nil {
		return true
	}
	switch b.typ {
	case ILList:
		if len(b.inner) == 0 {
			return true
		}
	case ILDataPtrAdd, ILDataAdd, ILWrite:
		if b.param == 0 {
			return true
		}
	}

	return false
}

// Prune removes all No-Operations that the Optimize step may have produced.
// For example dataadd(0) or dataptradd(0)
// This must be done depth first, to ensure that parent nodes will be pruned
// after leaf nodes.
// TODO: Parallelize
func (b *ILBlock) Prune() {

	if len(b.inner) == 0 {
		return
	}

	oldinner := b.inner
	b.inner = make([]*ILBlock, 0)
	for _, ib := range oldinner {
		ib.Prune()
		if !ib.isPruneable() {
			b.Append(ib)
		}
	}
}

type voverlay struct {
	ptrOff int
	header *ILBlock
	vec    *ILBlock
	footer *ILBlock
}

func (c *voverlay) dataadd(value byte) {
	if c.ptrOff < 0 {
		// must extend negatively
		newLen := (-c.ptrOff) + len(c.vec.vec)
		newV := make([]byte, newLen, newLen*2)

		copy(newV[(-c.ptrOff):], c.vec.vec)
		c.vec.vec = newV

		// place back to 0
		c.header.param += int64(c.ptrOff)
		c.ptrOff = 0
		c.footer.param = int64(c.ptrOff)

	} else if c.ptrOff >= len(c.vec.vec) {
		// must extend positive
		if c.ptrOff < cap(c.vec.vec) {
			c.vec.vec = c.vec.vec[:c.ptrOff+1]
		} else {
			newV := make([]byte, c.ptrOff+1, (c.ptrOff+1)*2)
			copy(newV, c.vec.vec)
			c.vec.vec = newV
		}
	}

	c.vec.vec[c.ptrOff] += value
}

func (c *voverlay) dataptradd(delta int64) {
	c.ptrOff += int(delta)
	c.footer.param = int64(c.ptrOff)
}

func (b *ILBlock) Vectorize() {
	// for long blocks that don't print, aggregate their data deltas
	// and dataptr moves into the following:
	// * dataptr move
	// * data vector deltas apply
	// * dataptr move

	// base condition
	if b.typ != ILList && b.typ != ILLoop {
		return
	}

	var wg sync.WaitGroup

	// rip through inner ILBlocks
	oldinner := b.inner
	b.inner = make([]*ILBlock, 0)

	var lastVec *voverlay
	for _, ib := range oldinner {
		switch ib.typ {
		case ILList, ILLoop:
			if lastVec != nil {
				lastVec = nil
			}
			b.Append(ib)
			wg.Add(1)
			go func(wg *sync.WaitGroup, ib *ILBlock) {
				ib.Vectorize()
				wg.Done()
			}(&wg, ib)
		case ILRead, ILWrite:
			if lastVec != nil {
				lastVec = nil
			}
			b.Append(ib)
		case ILDataAdd:
			if lastVec == nil {
				lastVec = &voverlay{
					header: &ILBlock{
						typ: ILDataPtrAdd,
					},
					vec: &ILBlock{
						typ: ILDataAddVector,
						vec: make([]byte, 0),
					},
					footer: &ILBlock{
						typ: ILDataPtrAdd,
					},
				}
				b.Append(lastVec.header)
				b.Append(lastVec.vec)
				b.Append(lastVec.footer)
			}
			lastVec.dataadd(byte(ib.param))
		case ILDataPtrAdd:
			if lastVec != nil {
				lastVec.dataptradd(ib.param)
			} else {
				b.Append(ib)
			}
		case ILDataAddVector:
			b.Append(ib)
		}
	}
	wg.Wait()
}
