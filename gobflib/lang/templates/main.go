package main

import "bytes"
import "os"
{{ if .ProfilingEnabled }}
import (
	"fmt"
	"time"
)
{{ end }}

var data []byte
var datap int
{{ if .ProfilingEnabled }}
var datapMax int
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
	fmt.Fprintln(os.Stderr, "Runtime:     ", time.Now().Sub(start))
	fmt.Fprintln(os.Stderr, "Data Ptr:    ", datap)
	fmt.Fprintln(os.Stderr, "Data Ptr Max:", datapMax)
	fmt.Fprintln(os.Stderr, "Data Length: ", len(data))
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
		newdata := make([]byte, len(data)*2)
		copy(newdata, data)
		data = newdata
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
	if (datap + len(vec) - 1) >= len(data) {
		newdata := make([]byte, len(data)*2)
		copy(newdata, data)
		data = newdata
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

func main() {
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
