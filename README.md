[![Go Reference](https://pkg.go.dev/badge/github.com/linux4life798/gobf/gobflib.svg)](https://pkg.go.dev/github.com/linux4life798/gobf/gobflib)
[![Go Build/Test](https://github.com/linux4life798/gobf/actions/workflows/go.yml/badge.svg)](https://github.com/linux4life798/gobf/actions/workflows/go.yml)
[![CodeQL](https://github.com/linux4life798/gobf/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/linux4life798/gobf/actions/workflows/codeql-analysis.yml)

# GoBF
This is a [BF][wikipedia-bf] interpreter and optimizing compiler written in Go.

GoBF can simply run your BF program or compile it to a binary to run later.

## Obligatory Install Line
```sh
go get github.com/linux4life798/gobf
```

## Usage
The commandline program currently supports `compile`, `gengo`,
`run`, and `dumpil` actions.

Give it a try!
```sh
go get github.com/linux4life798/gobf

wget https://github.com/erikdubbelboer/brainfuck-jit/raw/master/mandelbrot.bf
gobf compile mandelbrot.bf
./mandelbrot
```

Note that the `run` command will simply interpret the BF program in-place,
thus the performance will be as-is. Please use the `compile` to generate
an optimized program.

Please see `gobf --help` for more fun options!

## Optimization
The generated code optimizer reduces redundant and repetitive commands,
like data pointer moves or incrementing a data cell.
It coalesces multiple moves or data cell changes into one operation.
Due to BF's repetitive nature, this typically increases the BF program's
performance dramatically. All of the interesting optimization stuff
is in [gobflib/il](gobflib/il) package.

Recent work has added some pattern and vectorization based optimizations,
but these have not been fully calibrated yet.

To try the zero pattern optimization, invoke gobf in the following manor:
```sh
gobf -O zero compile mandelbrot.bf
```

[wikipedia-bf]: https://en.wikipedia.org/wiki/Brainfuck
