package main

import (
	"fmt"
	"os"
	"github.com/hcyang1106/simple-linker/pkg/utils"
	"github.com/hcyang1106/simple-linker/pkg/linker"
)

// functions handle errs themselves
func main() {
	ctx := linker.NewContext()
	parseArgs(ctx)

	fmt.Println(ctx.Args.Output)
	os.Exit(0)

	if len(os.Args) < 2 {
		utils.Fatal("Invalid input arguments")
	}

	file := linker.NewFile(os.Args[1])
	linker.MustHaveMagic(file.Content)

	inputFile := linker.NewInputFile(file)
	utils.Assert(len(inputFile.ElfSecHdrs) == 11)

	// first section of elf file is usually empty
	for _, shdr := range inputFile.ElfSecHdrs {
		fmt.Println(linker.ElfGetName(inputFile.ShStrTab, shdr.Name))
	}

	inputFile.ParseSymTab()
	utils.Assert(inputFile.FirstGlobal == 10)
	utils.Assert(len(inputFile.ElfSyms) == 12)

	// SymStrTab stores symbol names (array), and the Name field
	// of symbol element stores the offset
	for _, sym := range inputFile.ElfSyms {
		fmt.Println(linker.ElfGetName(inputFile.SymStrTab, sym.Name))
	}
}

func addDashes(option string) []string {
	res := []string{}

	if (len(option) == 1) {
		res = append(res, "-" + option)
	} else {
		res = append(res, "-" + option, "--" + option)
	}

	return res
}

// usage: readFlag("help")
// or readFlag("o")
func readFlag(option string, currStr string) bool {
	for _, opt := range addDashes(option) {
		if currStr == opt {
			return true
		}
	}

	return false
}

func readOpt(option string, currStr string) bool {
	for _, opt := range addDashes(option) {
		if currStr == opt {
			return true
		}
	}

	return false
}

func parseArgs(ctx *linker.Context) []string {
	args := os.Args

	remaining := make([]string, 0)
	for len(args) > 0 {
		if readFlag("help", args[0]) {
			fmt.Printf("usage: %s [options] files...\n", os.Args[0])
			os.Exit(0)
		} else if readOpt("o", args[0]) || readOpt("output", args[0]) {
			args = args[1:]
			if len(args) == 0 {
				utils.Fatal("No output filename specified")
			}
			ctx.Args.Output = args[0]
		} else {
			remaining = append(remaining, args[0])
		}
		args = args[1:]
	}

	return remaining
}
