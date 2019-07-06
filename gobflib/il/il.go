//go:generate stringer -type=ILBlockType

package il

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

type ILBlockType byte

const (
	ILList ILBlockType = iota
	ILLoop
	ILDataPtrAdd
	ILDataAdd
	ILDataSet
	ILRead
	ILWrite
	ILDataAddVector
	ILDataAddLinVector // param is offset of vector
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

func (b *ILBlock) ResetInner(size int) {
	if size < 0 {
		size = len(b.inner)
	}
	b.inner = make([]*ILBlock, 0, size)
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
	fmt.Fprintf(out, "%*s--------------------------\n", indent*indentWidth, "")
	if b == nil {
		fmt.Fprintf(out, "%*s<nil>\n", indent*indentWidth, "")
		return
	}
	fmt.Fprintf(out, "%*s| %-12v |", indent*indentWidth, "", b.typ)
	switch b.typ {
	case ILList, ILLoop:
	case ILDataAdd, ILDataPtrAdd, ILDataSet:
		fmt.Fprintf(out, " param=%v |", b.param)
	case ILDataAddVector:
		fmt.Fprintf(out, " vec=%v |", b.vec)
		vc, oc := b.vectorCost()
		fmt.Fprintf(out, " vcost=%d ocost=%d", vc, oc)
	case ILDataAddLinVector:
		fmt.Fprintf(out, " off=%v |", b.param)
		fmt.Fprintf(out, " vec=%v |", b.vec)
		vc, oc := b.vectorCost()
		fmt.Fprintf(out, " vcost=%d ocost=%d", vc, oc)
	}
	fmt.Fprintf(out, "\n")
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

// Compress combines adjacent same type ILBlocks that have repeat parameters
//
// This is one case, where multiple Compress/Prune cycles are necessary.
// This can really only happen after a VectorBalance step.
//
// ILDataAdd    -1
// ILDataPtrAdd  0
// ILDataAdd     1
//
// Can this happen more than once?
//
// ILDataAdd    -1
// ILDataPtrAdd  0
// ILDataAdd     1
func (b *ILBlock) Compress() int {
	var count int64

	// base condition
	if b.typ != ILList && b.typ != ILLoop {
		return int(count)
	}

	var oldinner []*ILBlock

	/* This step expands ILLists elements into the parent ILBlock */
	oldinner = b.GetInner()
	b.ResetInner(-1)
	for _, ib := range oldinner {
		if ib.typ == ILList {
			b.Append(ib.inner...)
			count += int64(len(ib.inner))
		} else {
			b.Append(ib)
		}
	}

	/* This step combines similar consecutive ILBlock types */
	var wg sync.WaitGroup
	oldinner = b.GetInner()
	b.ResetInner(-1)
	var lastb *ILBlock
	for _, ib := range oldinner {
		switch ib.typ {
		case ILList, ILLoop:
			b.Append(ib)
			wg.Add(1)
			go func(wg *sync.WaitGroup, ib *ILBlock) {
				c := ib.Compress()
				atomic.AddInt64(&count, int64(c))
				wg.Done()
			}(&wg, ib)
			lastb = nil
		case ILDataPtrAdd, ILWrite:
			/* Combine DataPtrAdds or WriteBs */
			if lastb != nil && lastb.typ == ib.typ {
				// combine with previous run
				lastb.param += ib.param
				atomic.AddInt64(&count, 1)
			} else {
				// start next run
				b.Append(ib)
				lastb = ib
			}
		case ILDataAdd:
			/* Combine DataAdds, DataPtrAdds, and WriteBs */
			if lastb != nil {
				switch lastb.typ {
				case ILDataAdd, ILDataSet:
					// combine with previous DataAdd or DataSet
					lastb.param += ib.param
					atomic.AddInt64(&count, 1)
				default:
					b.Append(ib)
					lastb = ib
				}
			} else {
				// start next run
				b.Append(ib)
				lastb = ib
			}
		case ILDataSet:
			/* Overrive a previous ILDataSet or ILDataAdd(interesting eh?) */
			if lastb != nil {
				switch lastb.typ {
				case ILDataSet, ILDataAdd:
					// combine with previous run
					lastb.typ = ILDataSet // override a previous DataAdd
					lastb.param = ib.param
					atomic.AddInt64(&count, 1)
				default:
					b.Append(ib)
					lastb = ib
				}
			} else {
				// start next run
				b.Append(ib)
				lastb = ib
			}
		case ILDataAddVector:
			fallthrough
		case ILDataAddLinVector:
			fallthrough
		default:
			b.Append(ib)
			lastb = nil
		}
	}

	wg.Wait()

	return int(count)
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
func (b *ILBlock) Prune() int {
	var count int

	if len(b.inner) == 0 {
		return count
	}

	oldinner := b.inner
	b.inner = make([]*ILBlock, 0)
	for _, ib := range oldinner {
		count += ib.Prune()
		if !ib.isPruneable() {
			b.Append(ib)
		} else {
			count++
		}
	}

	return count
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

func (b *ILBlock) vectorCost() (vcost, icost int) {
	const datapaddCost = 1 + 1 + 1         // add, check <0, check readjust
	const dataaddvecStaticCost = 1 + 1 + 1 // check readjust, slice, bound check
	if b.typ != ILDataAddVector && b.typ != ILDataAddLinVector {
		return -1, -1
	}

	// Vector Static Cost
	vcost += dataaddvecStaticCost
	// Vector Dynamic Cost
	vcost += len(b.vec)

	for _, v := range b.vec {
		if v != 0 {
			// Independent Dynamic Cost
			icost += datapaddCost
		}
	}

	return
}

func (b *ILBlock) Vectorize() int {
	// for long blocks that don't print, aggregate their data deltas
	// and dataptr moves into the following:
	// * dataptr move
	// * data vector deltas apply
	// * dataptr move

	var count int64

	// base condition
	if b.typ != ILList && b.typ != ILLoop {
		return int(count)
	}

	var wg sync.WaitGroup
	var oldinner []*ILBlock

	// rip through inner ILBlocks
	oldinner = b.GetInner()
	b.ResetInner(-1)

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
				c := ib.Vectorize()
				atomic.AddInt64(&count, int64(c))
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
				atomic.AddInt64(&count, 1)
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

	return int(count)
}

// VectorBalance runs after Vectorizing and determines the runtime
// cost of keeping vectorized adds as compared to having independent operations.
// If the cost is higher to have vectorized operations, they are split up.
func (b *ILBlock) VectorBalance() int {
	var count int64

	// base condition
	if b.typ != ILList && b.typ != ILLoop {
		return int(count)
	}

	var wg sync.WaitGroup

	// rip through inner ILBlocks
	oldinner := b.inner
	b.inner = make([]*ILBlock, 0)

	for _, ib := range oldinner {
		switch ib.typ {
		case ILList, ILLoop:
			b.Append(ib)
			wg.Add(1)
			go func(wg *sync.WaitGroup, ib *ILBlock) {
				c := ib.VectorBalance()
				atomic.AddInt64(&count, int64(c))
				wg.Done()
			}(&wg, ib)
		case ILDataAddVector:
			vcost, ocost := ib.vectorCost()
			if vcost > ocost {
				// Break It Up
				for _, v := range ib.vec {
					b.Append(&ILBlock{
						typ:   ILDataAdd,
						param: int64(v),
					})
					b.Append(&ILBlock{
						typ:   ILDataPtrAdd,
						param: 1,
					})
				}

				// Add corrective dataptr. This should be the exact inverse
				// data ptr value of the next ILBlock.
				// This will be combined with original footer
				// using an additional Optimize step and remove when
				// after a Prune.
				b.Append(&ILBlock{
					typ:   ILDataPtrAdd,
					param: int64(-len(ib.vec)),
				})
				atomic.AddInt64(&count, 1)
			} else {
				b.Append(ib)
			}
		case ILRead, ILWrite, ILDataAdd, ILDataPtrAdd:
			fallthrough
		default:
			b.Append(ib)
		}
	}
	wg.Wait()

	return int(count)
}

type PatternReplacer func(b *ILBlock) []*ILBlock

// Compatible with DataAdd and DataAddVector
func PatternReplaceZero(b *ILBlock) []*ILBlock {
	if b.typ == ILLoop {
		if len(b.inner) != 1 {
			return nil
		}

		switch loop := b.inner[0]; loop.typ {
		case ILDataAdd:
			// data add with -1 (0xFF)
			if loop.param != -1 {
				return nil
			}
		case ILDataAddVector:
			// vector with one element -1 (0xFF)
			if len(loop.vec) != 1 || loop.vec[0] != 0xff {
				return nil
			}
		default:
			return nil
		}

		return []*ILBlock{
			&ILBlock{
				typ:   ILDataSet,
				param: 0,
			},
		}
	} else if b.typ == ILDataAddLinVector {
		if len(b.vec) != 1 || b.vec[0] != 0xFF || b.param != 0 {
			return nil
		}
		return []*ILBlock{
			&ILBlock{
				typ:   ILDataSet,
				param: 0,
			},
		}
	}
	return nil
}

func PatternReplaceLinearVector(b *ILBlock) []*ILBlock {
	if b.typ != ILLoop {
		return nil
	}

	var addvec *ILBlock
	var off int64

	if len(b.inner) == 1 {
		if b.inner[0].typ != ILDataAddVector {
			return nil
		}
		addvec = b.inner[0]
	} else if len(b.inner) == 3 {
		if b.inner[0].typ != ILDataPtrAdd ||
			b.inner[1].typ != ILDataAddVector ||
			b.inner[2].typ != ILDataPtrAdd {
			return nil
		}
		addvec = b.inner[1]
		off = b.inner[0].param

		if !(b.inner[0].param <= 0 && b.inner[2].param >= 0) {
			return nil
		}

		if b.inner[0].param != -b.inner[2].param {
			return nil
		}

		if len(addvec.vec) <= int(b.inner[2].param) || addvec.vec[b.inner[2].param] != 0xFF {
			return nil
		}
	} else {
		return nil
	}

	return []*ILBlock{
		&ILBlock{
			typ:   ILDataAddLinVector,
			param: off,
			vec:   addvec.vec,
		},
	}
}

func (b *ILBlock) PatternReplace(replacers ...PatternReplacer) int {
	var count int64

	// Try all the replacer. If one matches and returns
	// a set of replacement instructions, wrap them in an ILList
	// replace the current ILBlock.
	for _, replacer := range replacers {
		if rep := replacer(b); rep != nil {
			atomic.AddInt64(&count, 1)
			// wrap it in an ILList
			b.typ = ILList
			b.inner = rep
			break
		}
	}

	// rip through inner ILBlocks
	var wg sync.WaitGroup
	for _, ib := range b.GetInner() {
		// Recursively search for sub-matches.
		// We allow an already matched and replaced ILBlock to be searched
		// again.
		wg.Add(1)
		go func(wg *sync.WaitGroup, ib *ILBlock) {
			c := ib.PatternReplace(replacers...)
			atomic.AddInt64(&count, int64(c))
			wg.Done()
		}(&wg, ib)
	}
	wg.Wait()

	return int(count)
}

// BlockCount counts the total number of ILBlocks in the tree b
func (b *ILBlock) BlockCount() int {
	var count int64 = 1

	var wg sync.WaitGroup
	for _, ib := range b.GetInner() {
		switch ib.typ {
		case ILList, ILLoop:
			wg.Add(1)
			go func(wg *sync.WaitGroup, ib *ILBlock) {
				c := int64(ib.BlockCount())
				atomic.AddInt64(&count, int64(c))
				wg.Done()
			}(&wg, ib)
		default:
			atomic.AddInt64(&count, 1)
		}
	}
	wg.Wait()

	return int(count)
}

// You can't really predict the max data depth, since
// you can use a loop to skip forward one at a time.
// For example, this program sets the first two cells
// to 1, rewinds the ptr back, has a loop find the last cell,
// and then moves two places past.
// +>+><<[>]>>
func (b *ILBlock) PredictMaxDataSize() int {
	var deltaMax int64
	var delta int64

	// base condition
	if b.typ == ILDataPtrAdd {
		if b.param > 0 {
			deltaMax = b.param
		}
		return int(deltaMax)
	}
	if b.typ != ILList && b.typ != ILLoop {
		return int(deltaMax)
	}

	// var wg sync.WaitGroup

	for _, ib := range b.inner {
		switch ib.typ {
		case ILDataPtrAdd:
			delta += ib.param
			if delta > deltaMax {
				deltaMax = delta
			}
		case ILList, ILLoop:
			// wg.Add(1)
			// go func(wg *sync.WaitGroup, ib *ILBlock) {
			dMax := int64(ib.PredictMaxDataSize()) + delta
			if dMax > deltaMax {
				deltaMax = dMax
			}
			// atomic.AddInt64(&delta, int64(c))
			// wg.Done()
			// }(&wg, ib)
		case ILRead, ILWrite, ILDataAdd:
		case ILDataAddVector:
		}
	}
	// wg.Wait()

	return int(deltaMax)
}
