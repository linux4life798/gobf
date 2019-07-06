#!/bin/bash
# Craig Hesling
# This script generates a BF program that contains nested loops,
# each of which count down from 255 to 0. The inner-most operation
# is series of +'s in consecutive cells.
#
# The idea is to generate a hand crafet case, where vectorizing the inner-most
# operation yields great performance gains.


vector-content() {
    local indent=${1:-0}
    local ops=${2:-100}

    printf "%*s" $((indent*2)) ""
    for ((i = 0; i < $ops; i++)); do
        echo -n "+>"
    done
    echo


    printf "%*s" $((indent*2)) ""
    for ((i = 0; i < $ops; i++)); do
        echo -n "<"
    done
    echo
}

loop() {
    local indent=${1:-0}

    # Set max value by subtracting 1 from 0: set 255
    printf "%*s\n" $((indent*2)) "-"

    printf "%*s[->\n" $((indent*2)) ""

    printf "%*s" $((indent*2)) ""
    while IFS= read -r -s line; do
        printf "%*s\n" $((indent*2)) "$line"
    done
    echo

    printf "%*s<]\n" $((indent*2)) ""
}

echo "Craig Hesling"
echo

# 10 ops yielded a 28.6s     [vector] vs 1m28.26s difference
#  5 ops yielded a 31sec     [vector] vs 55s diff
#  3 ops yielded a 0m25.245s [vector] vs 0m31.317s
#  2 ops yielded a 0m24.190s [vector] vs 0m27.647s
#  1 ops yielded a 0m18.224s [vector] vs 0m19.120s
#vector-content 4 10 | loop 3 | loop 2 | loop 1 | loop 0

# vector-content 3 20 | loop 2 | loop 1 | loop 0
vector-content 2 20 | loop 1 | loop 0
# vector-content 1 20 | loop 0