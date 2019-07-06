#!/bin/bash
# Craig Hesling

# Generate vectorization's optimial test
./testprograms/gen-vector-test.bash > testprograms/vector-test.b

# Examples:
# https://github.com/fabianishere/brainfuck/tree/master/examples
# https://github.com/erikdubbelboer/brainfuck-jit

BENCHMARK=(
	https://github.com/fabianishere/brainfuck/raw/master/examples/hello.bf
	https://github.com/erikdubbelboer/brainfuck-jit/raw/master/mandelbrot.bf
#	Need to really zoom out your terminal
	https://github.com/fabianishere/brainfuck/raw/master/examples/mandelbrot/mandelbrot-titannic.bf
	https://github.com/fabianishere/brainfuck/raw/master/examples/mandelbrot/mandelbrot-huge.bf
	https://github.com/fabianishere/brainfuck/raw/master/examples/hanoi.bf
)

INTERACTIVE=(
	https://github.com/fabianishere/brainfuck/raw/master/examples/lost-kingdom.bf
	https://github.com/fabianishere/brainfuck/raw/master/examples/gameoflife.bf
)

if ! which wget &>/dev/null; then
	echo "Error - wget is not installed. This script required the wget tool." >&2
	exit 1
fi

mkdir -p testprograms/interactive

wget -c -P testprograms "${BENCHMARK[@]}"
wget -c -P testprograms/interactive "${INTERACTIVE[@]}"
