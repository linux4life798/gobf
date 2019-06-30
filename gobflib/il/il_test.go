package il

import "testing"

func TestILEqual(t *testing.T) {
	if !NewILBlock(ILList).Equal(NewILBlock(ILList)) {
		t.Error("Failed equate two blank ILLists")
	}

	chainAdd := func(b *ILBlock) {
		b1 := NewILBlock(ILList)
		b2 := NewILBlock(ILList)
		b3 := NewILBlock(ILList)

		add1 := NewILBlock(ILDataAdd)
		add2 := NewILBlock(ILDataAdd)
		add3 := NewILBlock(ILDataAdd)
		add4 := NewILBlock(ILDataAdd)
		add5 := NewILBlock(ILDataAdd)
		add2.param = 1
		add3.param = 2
		add4.param = 100
		add5.param = 3000

		b.Append(b1)
		b.Append(b2)
		b2.Append(b3)

		b.Append(add1)
		b1.Append(add2)
		b1.Append(add3)
		b2.Append(add4)
		b3.Append(add5)
	}

	il1 := NewILBlock(ILList)
	il2 := NewILBlock(ILList)

	chainAdd(il1)
	chainAdd(il1)
	chainAdd(il2)
	chainAdd(il2)

	if !il1.Equal(il2) {
		t.Error("Failed equate two multi-level trees")
	}

	il1 = NewILBlock(ILList)
	il2 = NewILBlock(ILList)

	chainAdd(il1)

	if il1.Equal(il2) {
		t.Error("Failed to detect differences between two trees")
	}
}

func TestILPrune(t *testing.T) {
	il := NewILBlock(ILList)
	il.Append(NewILBlock(ILDataAdd))
	il.Prune()

	if !il.Equal(NewILBlock(ILList)) {
		t.Error("Failed to prune a list with one 0 data add")
	}
}
