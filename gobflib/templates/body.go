package main

import "os"

var datap int
var data []byte

func writeb() {
	os.Stdout.Write(data[datap : datap+1])
}

func readb() {
	os.Stdin.Read(data[datap : datap+1])
}

func datapinc() {
	datap++
	if datap >= len(data) {
		newdata := make([]byte, len(data)*2)
		copy(newdata, data)
		data = newdata
	}
}

func datapdec() {
	if datap == 0 {
		panic("Data pointer is out of bounds")
	}
	datap--
}

func dataadd(delta byte) {
	data[datap]+=delta
}

func main() {
	data = make([]byte, {{ .InitialDataSize }})

	{{ .Body }}
}
