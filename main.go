package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/linux4life798/gobf/gobflib/lang"

	. "github.com/linux4life798/gobf/gobflib"
	"github.com/spf13/cobra"
)

const (
	defaultDataSize = 100000
	debugEnabled    = false
)

func BFRun(cmd *cobra.Command, args []string) {
	filename := args[0]
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file \"%s\": %v\n", filename, err)
		os.Exit(1)
	}
	defer f.Close()

	finfo, err := f.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stat file \"%s\": %v\n", filename, err)
		os.Exit(1)
	}

	fsize := finfo.Size()
	prgm := NewBFProgram(uint64(fsize), defaultDataSize)
	prgm.ReadCommands(f)
	if debugEnabled {
		fmt.Fprintf(os.Stderr, "Commands: ")
		prgm.PrintProgram(os.Stderr)
		fmt.Fprintf(os.Stderr, "\n")
	}
	if err := prgm.Run(); err != nil {
		fmt.Println(err)
	}
	if debugEnabled {
		fmt.Fprintln(os.Stderr, "Program terminated")
	}
}

func BFGenGo(cmd *cobra.Command, args []string) {
	filename := args[0]
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file \"%s\": %v\n", filename, err)
		os.Exit(1)
	}
	defer f.Close()

	finfo, err := f.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stat file \"%s\": %v\n", filename, err)
		os.Exit(1)
	}

	fsize := finfo.Size()
	prgm := NewBFProgram(uint64(fsize), defaultDataSize)
	prgm.ReadCommands(f)
	if debugEnabled {
		fmt.Fprintf(os.Stderr, "Commands: ")
		prgm.PrintProgram(os.Stderr)
		fmt.Fprintf(os.Stderr, "\n")
	}

	output := os.Stdout
	if len(args) > 1 {
		outputfilename := args[1]
		output, err = os.Create(outputfilename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open file \"%s\": %v\n", outputfilename, err)
			os.Exit(1)
		}
	}

	il := prgm.CreateILTree()
	il.Optimize()
	il.Prune()
	err = lang.ILBlockToGo(il, output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate Go: %v\n", err)
		os.Exit(1)
	}
}

func BFCompile(cmd *cobra.Command, args []string) {
	filename := args[0]
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file \"%s\": %v\n", filename, err)
		os.Exit(1)
	}
	defer f.Close()

	finfo, err := f.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stat file \"%s\": %v\n", filename, err)
		os.Exit(1)
	}

	fsize := finfo.Size()
	prgm := NewBFProgram(uint64(fsize), defaultDataSize)
	prgm.ReadCommands(f)
	if debugEnabled {
		fmt.Fprintf(os.Stderr, "Commands: ")
		prgm.PrintProgram(os.Stderr)
		fmt.Fprintf(os.Stderr, "\n")
	}

	tempdir, err := ioutil.TempDir("", "gobfcompile")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	gofile, err := os.Create(tempdir + "/main.go")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp file: %v\n", err)
		os.Exit(1)
	}

	il := prgm.CreateILTree()
	il.Optimize()
	il.Prune()
	err = lang.ILBlockToGo(il, gofile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate Go: %v\n", err)
		os.Exit(1)
	}

	var outputfilename string

	if len(args) > 1 {
		outputfilename = args[1]
	} else {
		basename := path.Base(filename)
		basenameparts := strings.Split(basename, ".")
		basename = basenameparts[0]

		// Sanity check
		if basename == filename {
			fmt.Fprintf(os.Stderr, "Error - Asked to overwrite original input fil\n")
			os.Exit(1)
		}

		outputfilename = basename
	}

	gobuild := exec.Command("go", "build", "-o", outputfilename, gofile.Name())
	err = gobuild.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build binary from Go: %v\n", err)
		os.Exit(1)
	}
	err = gobuild.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build binary from Go: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	var cmdRun = &cobra.Command{
		Use:   "run [bf file]",
		Short: "Run the given bf file",
		Long:  `This will evoke the interpreter for a specified bf text file`,
		Args:  cobra.MinimumNArgs(1),
		Run:   BFRun,
	}
	var cmdGenGo = &cobra.Command{
		Use:   "gengo [bf file] [output go file]",
		Short: "Generate a Go representation of the given bf file",
		Long:  `This will parse a given bf text file and generate equivalent Go code`,
		Args:  cobra.MinimumNArgs(1),
		Run:   BFGenGo,
	}
	var cmdCompile = &cobra.Command{
		Use:   "compile [bf file]",
		Short: "Compile the given bf file to a binary",
		Long:  `This will parse a given bf text file and generate equivalent binary program`,
		Args:  cobra.MinimumNArgs(1),
		Run:   BFCompile,
	}

	var rootCmd = &cobra.Command{Use: "gobf"}
	rootCmd.AddCommand(cmdRun)
	rootCmd.AddCommand(cmdGenGo)
	rootCmd.AddCommand(cmdCompile)
	rootCmd.Execute()
}
