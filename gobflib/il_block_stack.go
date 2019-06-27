package gobflib

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
