package main

import (
	"fmt"
	"github.com/hcyang1106/simple-linker/pkg/linker"
	"github.com/hcyang1106/simple-linker/pkg/utils"
	"os"
	"strings"

	//"strings"
)

var version string

// functions handle errs themselves
func main() {
	ctx := linker.NewContext()
	remaining := ctx.ParseArgs(ctx, version)

	// if machine type not specified, find it in obj file
	if ctx.Args.Machine == linker.MachineTypeNone {
		for _, filename := range remaining {
			if strings.HasPrefix(filename, "-") {
				continue
			}
			// obj file
			file := linker.NewFile(filename)
			mType := linker.GetMachineTypeFromContent(file.Content)
			if mType != linker.MachineTypeNone {
				ctx.Args.Machine = mType
				break
			}
		}
	}
	if ctx.Args.Machine != linker.MachineTypeRISCV64 {
		utils.Fatal("Unsupported machine type...")
	}

	ctx.FillInObjFiles(remaining) // remaining contains specific libraries or obj files

	fmt.Println(len(ctx.Args.ObjFiles))

	linker.MarkLiveObjects(ctx)
	fmt.Println("symbol count", len(ctx.SymbolMap))
	linker.ClearSymbolsAndFiles(ctx)
	fmt.Println("symbol count", len(ctx.SymbolMap))
	fmt.Println(len(ctx.Args.ObjFiles))

	for key, value := range ctx.SymbolMap {
		if key == "puts" {
			fmt.Println(value.File.File.Name)
		}
	}
	os.Exit(0)

	//if len(os.Args) < 2 {
	//	utils.Fatal("Invalid input arguments")
	//}
	//
	//file := linker.NewFile(os.Args[1])
	//linker.MustHaveMagic(file.Content)
	//
	//inputFile := linker.NewObjectFile(file)
	//utils.Assert(len(inputFile.ElfSecHdrs) == 11)
	//
	//// first section of elf file is usually empty
	//for _, shdr := range inputFile.ElfSecHdrs {
	//	fmt.Println(linker.ElfGetName(inputFile.ShStrTab, shdr.Name))
	//}
	//
	//inputFile.ParseSymTab()
	//utils.Assert(inputFile.FirstGlobal == 10)
	//utils.Assert(len(inputFile.ElfSyms) == 12)
	//
	//// SymStrTab stores symbol names (array), and the Name field
	//// of symbol element stores the offset
	//for _, sym := range inputFile.ElfSyms {
	//	fmt.Println(linker.ElfGetName(inputFile.SymStrTab, sym.Name))
	//}
}
