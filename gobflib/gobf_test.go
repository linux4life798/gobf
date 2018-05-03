package gobflib

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"testing"
)

type testanspair struct {
	name      string
	cmds      string
	input     []byte
	output    []byte
	testprint bool
}

var tests = []testanspair{
	testanspair{
		name:      "No program",
		cmds:      "",
		input:     []byte(""),
		output:    []byte(""),
		testprint: true,
	},
	testanspair{
		name:      "No program with comments and bad characters",
		cmds:      "\n # ignore <.>[]+- everything here\nNothing to run here  \n#++++++++++++++++++++++++++++++++++++++++++.",
		input:     []byte(""),
		output:    []byte(""),
		testprint: false,
	},
	testanspair{
		name:      "Echo four bytes manually",
		cmds:      ",>,>,>,<<<.>.>.>.",
		input:     []byte("abcd"),
		output:    []byte("abcd"),
		testprint: true,
	},
	testanspair{
		name:      "Echo four bytes using loop",
		cmds:      ",>,>,>,<<<[.>]",
		input:     []byte("abcd"),
		output:    []byte("abcd"),
		testprint: true,
	},
	testanspair{
		name:      "Print four stars",
		cmds:      "++++++++++++++++++++++++++++++++---+++++++++++++.... # ignore <.>[]+- everything here",
		input:     []byte{},
		output:    []byte("****"),
		testprint: false,
	},
	testanspair{
		name:      "Test PrintCommands",
		cmds:      "++++++++++++++++++++++++++++++++---+++++++++++++....[-]><",
		input:     []byte{},
		output:    []byte("****"),
		testprint: true,
	},
}

func GenerateNullReader() io.Reader {
	return bytes.NewReader([]byte{})
}

func TestPrintStar(t *testing.T) {
	bfcmds := strings.NewReader("++++++++++++++++++++++++++++++++++++++++++.... # ignore <.>[]+- everything here")
	answerBuffer := []byte("****")
	input := GenerateNullReader()
	output := bytes.NewBuffer([]byte{})

	prgm := NewIOBFProgram(1, 1, input, output)
	prgm.ReadCommands(bfcmds)
	err := prgm.Run()
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(output.Bytes(), answerBuffer) != 0 {
		t.Log("answer bytes:", answerBuffer)
		t.Log("output bytes:", output.Bytes())
		t.Fatal("Output does not match expected output")
	}
}

func RunTest(t *testing.T, tpair *testanspair) {
	bfcmds := strings.NewReader(tpair.cmds)
	input := bytes.NewReader(tpair.input)
	output := bytes.NewBuffer([]byte{})

	// Create program context and parse commands
	prgm := NewIOBFProgram(0, 0, input, output)
	prgm.ReadCommands(bfcmds)

	// For the sake of testing Clone
	prgm = prgm.Clone()

	// Run the cloned program
	if err := prgm.Run(); err != nil {
		t.Fatal(err)
	}

	// Check the program output
	if bytes.Compare(output.Bytes(), tpair.output) != 0 {
		t.Log("answer bytes:", tpair.output, string(tpair.output))
		t.Log("output bytes:", output.Bytes(), string(output.Bytes()))
		t.Fatal("Output does not match expected output")
	}

	// Test the PrintProgram function
	if tpair.testprint {
		outb := bytes.NewBuffer([]byte{})
		prgm.PrintProgram(outb)
		if bytes.Compare(outb.Bytes(), []byte(tpair.cmds)) != 0 {
			t.Log("printed bytes:", string(outb.Bytes()))
			t.Log("output bytes:", tpair.cmds)
			t.Fatal("Printed commands does not match inputted commands")
		}
	}
}

func TestTable(t *testing.T) {
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			RunTest(t, &tests[i])
		})
	}
}

func BenchmarkParsingSource(b *testing.B) {
	cmds, err := ioutil.ReadFile("testprograms/helloworld.b")
	if err != nil {
		b.Fatal(err)
	}
	// We don't want os read latency -- so read in entire file first
	cmdsreader := bytes.NewReader(cmds)
	var prgm *BFProgram

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := cmdsreader.Seek(0, io.SeekStart)
		if err != nil {
			b.Fatal(err)
		}
		prgm = NewBFProgram(uint64(cmdsreader.Len()), 0)
		prgm.ReadCommands(cmdsreader)
	}

	runtime.KeepAlive(prgm)
}

func BenchmarkRunningHelloWorld(b *testing.B) {
	cmdsfile, err := os.Open("testprograms/helloworld.b")
	if err != nil {
		b.Fatal(err)
	}
	defer cmdsfile.Close()
	input := bytes.NewReader([]byte{})

	prgm := NewIOBFProgram(0, 0, input, ioutil.Discard)
	prgm.ReadCommands(cmdsfile)

	prgms := make([]*BFProgram, b.N)
	for i := range prgms {
		prgms[i] = prgm.Clone()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := prgms[i].Run(); err != nil {
			b.Fatal(err)
		}
	}

	runtime.KeepAlive(prgm)
}

func BenchmarkRunningPrintStar(b *testing.B) {
	cmdsfile, err := os.Open("testprograms/printstar.b")
	if err != nil {
		b.Fatal(err)
	}
	defer cmdsfile.Close()
	input := bytes.NewReader([]byte{})

	prgm := NewIOBFProgram(0, 0, input, ioutil.Discard)
	prgm.ReadCommands(cmdsfile)

	prgms := make([]*BFProgram, b.N)
	for i := range prgms {
		prgms[i] = prgm.Clone()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := prgms[i].Run(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadFileAndRunPrintStar(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmdsfile, err := os.Open("testprograms/printstar.b")
		if err != nil {
			b.Fatal(err)
		}
		input := bytes.NewReader([]byte{})

		prgm := NewIOBFProgram(0, 0, input, ioutil.Discard)
		prgm.ReadCommands(cmdsfile)
		cmdsfile.Close()
		if err := prgm.Run(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadFileAndRunHelloWorld(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmdsfile, err := os.Open("testprograms/printstar.b")
		if err != nil {
			b.Fatal(err)
		}
		input := bytes.NewReader([]byte{})

		prgm := NewIOBFProgram(0, 0, input, ioutil.Discard)
		prgm.ReadCommands(cmdsfile)
		cmdsfile.Close()
		if err := prgm.Run(); err != nil {
			b.Fatal(err)
		}
	}
}
