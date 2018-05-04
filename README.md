[![Godoc](https://godoc.org/github.com/linux4life798/gobf/gobflib?status.png)](https://godoc.org/github.com/linux4life798/gobf/gobflib)

# GoBF
This is a [BF][wikipedia-bf] interpreter and compiler written in Go.

GoBF can simply run your BF program or compile it to a binary to run later.

The commandline program currently supports `run`, `gengo`, and `compile`
actions.

The generated code optimizer reduces redundant and repetitive commands,
like data pointer moves or incrementing a data cell.
It coalesces multiple moves or data cell changes into one operation.
Due to BF's repetitive nature, this typically increases the BF program's
performance dramatically.

[wikipedia-bf]: https://en.wikipedia.org/wiki/Brainfuck