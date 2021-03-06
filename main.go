package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/linux4life798/gobf/gobflib/il"

	"github.com/linux4life798/gobf/gobflib/lang"

	. "github.com/linux4life798/gobf/gobflib"
	"github.com/spf13/cobra"
)

const (
	defaultDataSize = 100000
)

var debugEnabled *bool

func dprintf(format string, a ...interface{}) {
	if *debugEnabled {
		fmt.Printf(format+"\n", a...)
	}
}

func prepareIL(cmd *cobra.Command, bfinput io.Reader, bfinputsize int64) (*il.ILBlock, error) {
	flagCompress, _ := cmd.Flags().GetBool("compress")
	flagPrune, _ := cmd.Flags().GetBool("prune")
	flagVectorize, _ := cmd.Flags().GetBool("vectorize")
	flagFullVectorize, _ := cmd.Flags().GetBool("full-vectorize")
	if flagFullVectorize {
		flagVectorize = true
	}
	flagOpts, _ := cmd.Flags().GetStringSlice("optimize")
	var optimization = make(map[string]bool)
	for _, opt := range flagOpts {
		optimization[opt] = true
	}

	dprintf("Reading BF Program")
	prgm := NewBFProgram(uint64(bfinputsize), defaultDataSize)
	prgm.ReadCommands(bfinput)

	var compressCount int
	var pruneCount int
	var vectorizeCount int
	var vectorBalanceCount int
	var optimizationCount int

	dprintf("Generating IL Representation")
	iltree := prgm.CreateILTree()
	if flagCompress {
		dprintf("Compressing IL")
		compressCount += iltree.Compress()
	}
	if flagPrune {
		dprintf("Pruning IL")
		pruneCount += iltree.Prune()
	}
	if flagVectorize {
		dprintf("Vectoring IL")
		vectorizeCount = iltree.Vectorize()
		if !flagFullVectorize {
			dprintf("Rebalancing Vectorized IL")
			vectorBalanceCount = iltree.VectorBalance()
		}

		// ILDataAdd    -1
		// ILDataPtrAdd  0
		// ILDataAdd     1

		// prune possible datapadd(0) after vector replace
		dprintf("Pruning IL")
		pruneCount += iltree.Prune()
		dprintf("Compressing IL")
		compressCount += iltree.Compress()
		dprintf("Pruning IL")
		pruneCount += iltree.Prune()

		if count := iltree.Compress(); count > 0 {
			fmt.Println("# Error", count, "Additional Compresses Were Necessary!")
		}
		if count := iltree.Prune(); count > 0 {
			fmt.Println("# Error", count, "Additional Prune Were Necessary!")
		}
	}

	if optimization["lvec"] {
		dprintf("Vectorizing IL")
		iltree.Vectorize()
		pruneCount += iltree.Prune()
		compressCount += iltree.Compress()
		pruneCount += iltree.Prune()

		dprintf("Pattern Linear Vectorizing IL")
		optimizationCount = iltree.PatternReplace(il.PatternReplaceLinearVector)
		optimizationCount += iltree.Compress()
		optimizationCount += iltree.Prune()

		if !flagFullVectorize {
			dprintf("Rebalancing Vectorized IL")
			vectorBalanceCount = iltree.VectorBalance()
			dprintf("Pruning IL")
			pruneCount += iltree.Prune()
			dprintf("Compressing IL")
			compressCount += iltree.Compress()
			dprintf("Pruning IL")
			pruneCount += iltree.Prune()
		}
	}

	if optimization["zero"] {
		// TODO: Implement dataset vectoring for situations where lots
		//       of consecutive cells are set to 0
		dprintf("Pattern Zero Replacing IL")
		optimizationCount = iltree.PatternReplace(il.PatternReplaceZero)
		dprintf("Compressing IL")
		optimizationCount += iltree.Compress()
		dprintf("Pruning IL")
		optimizationCount += iltree.Prune()
	}

	if *debugEnabled {
		if flagCompress {
			fmt.Println("Compress Count:        ", compressCount)
		}
		if flagPrune {
			fmt.Println("Prune Count:           ", pruneCount)
		}
		if flagVectorize {
			fmt.Println("Vectorized Count:      ", vectorizeCount)
			if !flagFullVectorize {
				fmt.Println("Vectors Balance Count: ", vectorBalanceCount,
					"(", vectorizeCount-vectorBalanceCount, "remain )")
			}
		}
		if len(flagOpts) > 0 {
			fmt.Println("Optimization Count:    ", optimizationCount)
		}
		fmt.Println("Final Block Count:     ", iltree.BlockCount())
	}

	return iltree, nil
}

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
	if err := prgm.Run(); err != nil {
		fmt.Println(err)
	}
	if *debugEnabled {
		fmt.Fprintln(os.Stderr, "Program terminated")
	}
}

func BFGenGo(cmd *cobra.Command, args []string) {
	flagProfile, _ := cmd.Flags().GetBool("profile")
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

	il, err := prepareIL(cmd, f, finfo.Size())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read BF and/or optimize: %v\n", err)
		os.Exit(1)
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

	err = lang.ILBlockToGo(il, output, flagProfile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate Go: %v\n", err)
		os.Exit(1)
	}
}

func BFDumpIL(cmd *cobra.Command, args []string) {
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

	il, err := prepareIL(cmd, f, finfo.Size())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read BF and/or optimize: %v\n", err)
		os.Exit(1)
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

	il.Dump(output, 0)
}

func BFCompile(cmd *cobra.Command, args []string) {
	flagProfile, _ := cmd.Flags().GetBool("profile")

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

	il, err := prepareIL(cmd, f, finfo.Size())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read BF and/or optimize: %v\n", err)
		os.Exit(1)
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

	dprintf("Compiling IL")
	err, tempdir := lang.CompileIL(il, outputfilename, *debugEnabled, flagProfile)
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
		Use:   "run <bf file>",
		Short: "Run the given bf file",
		Long:  `This will evoke the interpreter for a specified bf text file`,
		Args:  cobra.MinimumNArgs(1),
		Run:   BFRun,
	}
	var cmdGenGo = &cobra.Command{
		Use:   "gengo <bf file> [output go file]",
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
		Use:   "compile <bf file> [output go file]",
		Short: "Compile the given bf file to a binary",
		Long:  `This will parse a given bf text file and generate equivalent binary program`,
		Args:  cobra.MinimumNArgs(1),
		Run:   BFCompile,
	}

	var rootCmd = &cobra.Command{Use: "gobf"}
	debugEnabled = rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().BoolP("profile", "p", false, "Enable output program self profiling. This will slow down runtime.")
	rootCmd.PersistentFlags().BoolP("compress", "C", true, "Enable collapsing of repeat commands")
	rootCmd.PersistentFlags().BoolP("prune", "P", true, "Enable pruning of dead commands")
	rootCmd.PersistentFlags().BoolP("vectorize", "V", false, "Enable vectorizing of commands in a block")
	rootCmd.PersistentFlags().BoolP("full-vectorize", "F", false, "Force full vectorization without deciding cost tradeoff")
	rootCmd.PersistentFlags().StringSliceP("optimize", "O", []string{}, "Enables particular optimizations")
	rootCmd.AddCommand(cmdRun)
	rootCmd.AddCommand(cmdGenGo)
	rootCmd.AddCommand(cmdDumpIL)
	rootCmd.AddCommand(cmdCompile)
	rootCmd.Execute()
}
