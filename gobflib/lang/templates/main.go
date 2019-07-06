package main

import (
	"bytes"
	"fmt"
	"os"
)
{{ if .ProfilingEnabled }}
import (
	"crypto/sha1"
	"time"
)
{{ end }}

var data []byte
var datap int
{{ if .ProfilingEnabled }}
var datapMax int
var dataExpansionCount int
var start time.Time

func profUpdateDatapMax(dp int) {
	if dp > datapMax {
		datapMax = dp
	}
}

func profProgramStart() {
	start = time.Now()
}

func profProgramEnd() {
	const space = 22
	fmt.Fprintf(os.Stderr, "%-*s %v\n", space, "Runtime:", time.Now().Sub(start))
	fmt.Fprintf(os.Stderr, "%-*s %v\n", space, "Data Ptr:", datap)
	fmt.Fprintf(os.Stderr, "%-*s %v\n", space, "Data Ptr Max:", datapMax)
	fmt.Fprintf(os.Stderr, "%-*s %v\n", space, "Data Expansion Count:", dataExpansionCount)
	fmt.Fprintf(os.Stderr, "%-*s %v\n", space, "Data Length:", len(data))
	fmt.Fprintf(os.Stderr, "%-*s %v\n", space, "Data:", data[:datapMax+1])
	h := sha1.New()
	h.Write(data[:datapMax+1])
	fmt.Fprintf(os.Stderr, "%-*s %x\n", space, "Data:", h.Sum(nil))
}
{{ end }}

func writeb(repeat int) {
	os.Stdout.Write(bytes.Repeat(data[datap : datap+1], repeat))
}

func readb() {
	os.Stdin.Read(data[datap : datap+1])
}

func datapadd(delta int) {
	datap += delta
	if datap < 0 {
		panic("Data pointer is out of bounds")
	}
	if datap >= len(data) {
		newdata := make([]byte, datap*2)
		copy(newdata, data)
		data = newdata
		{{ if .ProfilingEnabled }}
		dataExpansionCount++
		{{ end }}
	}

	{{ if .ProfilingEnabled }}
	profUpdateDatapMax(datap)
	{{ end }}
}

func dataadd(delta byte) {
	data[datap] += delta
}

func dataset(value byte) {
	data[datap] = value
}

func dataaddvector(vec []byte) {
	// need to check data allocation
	if l := datap + len(vec) - 1; l >= len(data)  {
		newdata := make([]byte, l*2)
		copy(newdata, data)
		data = newdata
		{{ if .ProfilingEnabled }}
		dataExpansionCount++
		{{ end }}
	}
	var d = data[datap:]
	_ = d[len(vec)-1]
	for i := range vec {
		d[i] += vec[i]
	}

	{{ if .ProfilingEnabled }}
	profUpdateDatapMax(datap + len(vec) - 1)
	{{ end }}
}

func errorHandler() {
	if r := recover(); r != nil {
		fmt.Fprintln(os.Stderr, "Error:", r)
		{{ if .ProfilingEnabled }}
		profProgramEnd()
		{{ end }}
		panic(r)
	}
}

func main() {
	defer errorHandler()

	data = make([]byte, {{ .InitialDataSize }})

	{{ if .ProfilingEnabled }}
	profProgramStart()
	{{ end }}

	{{ range .Body }}{{ . }}
	{{ end }}

	{{ if .ProfilingEnabled }}
	profProgramEnd()
	{{ end }}
}
