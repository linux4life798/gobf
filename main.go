package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/linux4life798/gobf/gobflib/lang"

	. "github.com/linux4life798/gobf/gobflib"
	"github.com/spf13/cobra"
)

const (
	defaultDataSize = 100000
)

var debugEnabled *bool

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
	if *debugEnabled {
		fmt.Fprintf(os.Stderr, "Commands: ")
		prgm.PrintProgram(os.Stderr)
		fmt.Fprintf(os.Stderr, "\n")
	}
	if err := prgm.Run(); err != nil {
		fmt.Println(err)
	}
	if *debugEnabled {
		fmt.Fprintln(os.Stderr, "Program terminated")
	}
}

func BFGenGo(cmd *cobra.Command, args []string) {
	flagOptimize, _ := cmd.Flags().GetBool("optimize")
	flagPrune, _ := cmd.Flags().GetBool("prune")
	flagVectorize, _ := cmd.Flags().GetBool("vectorize")

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
	if *debugEnabled {
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
	if flagOptimize {
		il.Optimize()
	}
	if flagPrune {
		il.Prune()
	}
	if flagVectorize {
		il.Vectorize()
	}
	// prune possible datapadd(0) after vector replace
	if flagOptimize {
		il.Optimize()
	}
	if flagPrune {
		il.Prune()
	}
	err = lang.ILBlockToGo(il, output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate Go: %v\n", err)
		os.Exit(1)
	}
}

func BFDumpIL(cmd *cobra.Command, args []string) {
	flagOptimize, _ := cmd.Flags().GetBool("optimize")
	flagPrune, _ := cmd.Flags().GetBool("prune")
	flagVectorize, _ := cmd.Flags().GetBool("vectorize")

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
	if *debugEnabled {
		fmt.Fprintf(os.Stderr, "Commands: ")
		prgm.PrintProgram(os.Stderr)
		fmt.Fprintf(os.Stderr, "\n")

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
	if flagOptimize {
		il.Optimize()
	}
	if flagPrune {
		il.Prune()
	}
	if flagVectorize {
		il.Vectorize()
	}
	// prune possible datapadd(0) after vector replace
	if flagOptimize {
		il.Optimize()
	}
	if flagPrune {
		il.Prune()
	}
	il.Dump(output, 0)
}

func BFCompile(cmd *cobra.Command, args []string) {
	flagOptimize, _ := cmd.Flags().GetBool("optimize")
	flagPrune, _ := cmd.Flags().GetBool("prune")
	flagVectorize, _ := cmd.Flags().GetBool("vectorize")

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

	il := prgm.CreateILTree()
	if flagOptimize {
		il.Optimize()
	}
	if flagPrune {
		il.Prune()
	}
	if flagVectorize {
		il.Vectorize()
	}
	// prune possible datapadd(0) after vector replace
	if flagOptimize {
		il.Optimize()
	}
	if flagPrune {
		il.Prune()
	}

	var outputfilename string
	// If no output file specified, use basename (without extension)
	// as the program binary.
	if len(args) > 1 {
		outputfilename = args[1]
	} else {
		basename := path.Base(filename)
		basenameparts := strings.Split(basename, ".")
		basename = basenameparts[0]

		// Sanity check
		if basename == filename {
			fmt.Fprintf(os.Stderr, "Error - Asked to overwrite original input file\n")
			os.Exit(1)
		}

		outputfilename = basename
	}

	err, tempdir := lang.CompileIL(il, outputfilename, *debugEnabled)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error - %v", err)
		os.Exit(2)
	}
	if *debugEnabled {
		fmt.Println("TempDir:", tempdir)
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
	var cmdDumpIL = &cobra.Command{
		Use:   "dumpil <bf file> [output go file]",
		Short: "Dumps a text representation of the Intermediate Language Tree",
		Long:  `This will parse the bf file, generate the intermediate tree, run the specified optimizations, and print the tree.`,
		Args:  cobra.MinimumNArgs(1),
		Run:   BFDumpIL,
	}
	var cmdCompile = &cobra.Command{
		Use:   "compile [bf file]",
		Short: "Compile the given bf file to a binary",
		Long:  `This will parse a given bf text file and generate equivalent binary program`,
		Args:  cobra.MinimumNArgs(1),
		Run:   BFCompile,
	}

	var rootCmd = &cobra.Command{Use: "gobf"}
	debugEnabled = rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().BoolP("optimize", "O", true, "Enable collapsing of repeat commands")
	rootCmd.PersistentFlags().BoolP("prune", "P", true, "Enable pruning of dead commands")
	rootCmd.PersistentFlags().BoolP("vectorize", "V", false, "Enable vectorizing of commands in a block")
	rootCmd.AddCommand(cmdRun)
	rootCmd.AddCommand(cmdGenGo)
	rootCmd.AddCommand(cmdDumpIL)
	rootCmd.AddCommand(cmdCompile)
	rootCmd.Execute()
}
