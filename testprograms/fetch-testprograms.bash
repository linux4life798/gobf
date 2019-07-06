#!/bin/bash
# Craig Hesling

# Move to the directory containing this script
root="$(dirname $(realpath "$BASH_SOURCE"))"
cd "$root"

# Generate vectorization's optimial test
./gen-vector-test.bash > vector-test.b

# Examples:
# https://github.com/fabianishere/brainfuck/tree/master/examples
# https://github.com/erikdubbelboer/brainfuck-jit

BENCHMARK=(
	https://github.com/fabianishere/brainfuck/raw/master/examples/hello.bf
	https://github.com/erikdubbelboer/brainfuck-jit/raw/master/mandelbrot.bf
	https://github.com/fabianishere/brainfuck/raw/master/examples/mandelbrot/mandelbrot-huge.bf
	https://github.com/fabianishere/brainfuck/raw/master/examples/hanoi.bf
)

BENCHMARK_LONG=(
#	Need to really zoom out your terminal
	https://github.com/fabianishere/brainfuck/raw/master/examples/mandelbrot/mandelbrot-titannic.bf
)

INTERACTIVE=(
	https://github.com/fabianishere/brainfuck/raw/master/examples/lost-kingdom.bf
	https://github.com/fabianishere/brainfuck/raw/master/examples/gameoflife.bf
)

if ! which wget &>/dev/null; then
	echo "Error - wget is not installed. This script required the wget tool." >&2
	exit 1
fi

mkdir -p interactive longrunning

wget -c "${BENCHMARK[@]}"
wget -c -P longrunning "${BENCHMARK_LONG[@]}"
wget -c -P interactive "${INTERACTIVE[@]}"
