package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
		fmt.Fprintf(os.Stderr, "Failed to stat file \"%s\": %\nv", filename, err)
		os.Exit(1)
	}

	fsize := finfo.Size()
	prgm := NewBFProgram(uint64(fsize), defaultDataSize)
	prgm.ReadCommands(f)
	if debugEnabled {
		prgm.PrintProgram()
	}
	prgm.Reset()
	if err := prgm.Run(); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Program terminated")
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

	var rootCmd = &cobra.Command{Use: "gobf"}
	rootCmd.AddCommand(cmdRun)
	rootCmd.Execute()
}
