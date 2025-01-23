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
	// remaining contains -l and obj files
	// later extract objs from -l and -L params
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
	ctx.CreateInternalFile()

	fmt.Println(len(ctx.Args.ObjFiles))

	linker.MarkLiveObjects(ctx)
	fmt.Println("symbol count", len(ctx.SymbolMap))
	linker.ClearSymbolsAndFiles(ctx) // after marking alive files, we delete unused files and symbols in context
	fmt.Println(len(ctx.Args.ObjFiles))
	fmt.Println("symbol count", len(ctx.SymbolMap))

	linker.ChangeMSecsSymbolsSection(ctx)
	linker.CreateSyntheticSections(ctx)

	fileSize := linker.GetFileSize(ctx)
	ctx.Buf = make([]byte, fileSize)
	file, err := os.OpenFile(ctx.Args.Output, os.O_RDWR | os.O_CREATE, 0777)
	utils.MustNo(err)

	// after creating the buf, we could write into buf
	for _, writer := range ctx.OutputWriters {
		writer.CopyBuf(ctx)
	}

	_, err = file.Write(ctx.Buf)
	utils.MustNo(err)

	os.Exit(0)
}
