package gobflib

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/linux4life798/gobf/gobflib/lang"
)

const vec = true

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

var testFiles = []string{}

func init() {
	var matches []string
	matches, _ = filepath.Glob("../testprograms/*.b")
	testFiles = append(testFiles, matches...)
	matches, _ = filepath.Glob("../testprograms/*.bf")
	testFiles = append(testFiles, matches...)
	// matches, _ = filepath.Glob("../testprograms/bench/*.b")
	// testFiles = append(testFiles, matches...)
	// matches, _ = filepath.Glob("../testprograms/bench/*.bf")
	// testFiles = append(testFiles, matches...)
}

// RunTableTest is designed to run sub-tests
func RunTableTest(t *testing.T, tpair *testanspair) {
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

// RunBenchCompile is designed to run sub-benchmarks of the compilation
// process
func RunBenchCompile(b *testing.B, tpair *testanspair, vectorize bool) {
	for i := 0; i < b.N; i++ {
		bfcmds := strings.NewReader(tpair.cmds)
		input := bytes.NewReader(tpair.input)
		output := bytes.NewBuffer([]byte{})

		// Create program context and parse commands
		prgm := NewIOBFProgram(0, 0, input, output)
		prgm.ReadCommands(bfcmds)

		ilb := prgm.CreateILTree()
		ilb.Compress()
		ilb.Prune()
		if vectorize {
			ilb.Vectorize()
			ilb.VectorBalance()
			ilb.Compress()
			ilb.Prune()
		}
		if err, _ := lang.CompileIL(ilb, "/dev/null", false, false); err != nil {
			b.Fatal(err)
		}
	}
}

// RunBenchExecute is designed to run sub-benchmarks of the
func RunBenchExecute(b *testing.B, tpair *testanspair, vectorize bool) {
	const outbin = "/tmp/gobflib_bench"
	bfcmds := strings.NewReader(tpair.cmds)
	input := bytes.NewReader(tpair.input)
	output := bytes.NewBuffer([]byte{})

	// Create program context and parse commands
	prgm := NewIOBFProgram(0, 0, input, output)
	prgm.ReadCommands(bfcmds)

	ilb := prgm.CreateILTree()
	ilb.Compress()
	ilb.Prune()
	if vectorize {
		ilb.Vectorize()
		ilb.VectorBalance()
		ilb.Compress()
		ilb.Prune()
	}
	if err, _ := lang.CompileIL(ilb, outbin, false, false); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(outbin)
		cmd.Stdout = ioutil.Discard
		if err := cmd.Run(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestTable(t *testing.T) {
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			RunTableTest(t, &tests[i])
		})
	}
}

func BenchmarkCompileTable(b *testing.B) {
	for i := range tests {
		b.Run(tests[i].name, func(b *testing.B) {
			RunBenchCompile(b, &tests[i], vec)
		})
	}
}

func BenchmarkExecuteTable(b *testing.B) {
	for i := range tests {
		b.Run(tests[i].name, func(b *testing.B) {
			RunBenchExecute(b, &tests[i], vec)
		})
	}
}

func BenchmarkCompileFiles(b *testing.B) {
	for _, fname := range testFiles {
		cmds, err := ioutil.ReadFile(fname)
		if err != nil {
			b.Fatal(err)
		}
		var t testanspair
		t.name = filepath.Base(filepath.Dir(fname)) + "/" + filepath.Base(fname)
		t.cmds = string(cmds)
		t.input = []byte{}
		b.Run(t.name, func(b *testing.B) {
			RunBenchCompile(b, &t, vec)
		})
	}
}

func BenchmarkExecuteFiles(b *testing.B) {
	for _, fname := range testFiles {
		cmds, err := ioutil.ReadFile(fname)
		if err != nil {
			b.Fatal(err)
		}
		var t testanspair
		t.name = filepath.Base(filepath.Dir(fname)) + "/" + filepath.Base(fname)
		t.cmds = string(cmds)
		t.input = []byte{}
		b.Run(t.name, func(b *testing.B) {
			RunBenchExecute(b, &t, vec)
		})
	}
}

func TestNoProgramIL(t *testing.T) {
	input := bytes.NewBuffer([]byte{})
	output := bytes.NewBuffer([]byte{})

	// Create program context and parse commands
	prgm := NewIOBFProgram(0, 0, input, output)
	il := prgm.CreateILTree()
	il.Compress()
	lang.ILBlockToGo(il, output, false)
}

func BenchmarkParsingSource(b *testing.B) {
	cmds, err := ioutil.ReadFile("../testprograms/helloworld.b")
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
	cmdsfile, err := os.Open("../testprograms/helloworld.b")
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
	cmdsfile, err := os.Open("../testprograms/printstar.b")
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
		cmdsfile, err := os.Open("../testprograms/printstar.b")
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
		cmdsfile, err := os.Open("../testprograms/helloworld.b")
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
