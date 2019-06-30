package main

import "os"
import "bytes"

var datap int
var data []byte

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
	}
}

func dataadd(delta byte) {
	data[datap] += delta
}

func dataaddvector(vec []byte) {
	var p = datap
	for i := range vec {
		data[p] += vec[i]
		p++
	}
}

func main() {
	data = make([]byte, {{ .InitialDataSize }})

	{{ range .Body }}{{ . }}
	{{ end }}
}
