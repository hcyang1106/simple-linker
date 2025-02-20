package main

import (
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

	linker.MarkLiveObjects(ctx)
	linker.ClearSymbolsAndFiles(ctx) // after marking alive files, we delete unused files and symbols in context

	// loop through all the symbols in file and reset related input section to fragment
	linker.ChangeMSecsSymbolsSection(ctx)

	// for shdr, ehdr, phdr, got
	// need to update size and offset, but before that outputwriters slice should be confirmed
	// also need to update ehdr fields
	linker.CreateSpecialWriters(ctx)
	// fragment offsets can only be calculated after frags are confirmed (as well as the merged section size)
	linker.UpdateFragmentOffsetAndMergedSectionSizeAlign(ctx)

	// since some input sections are set to nil
	linker.SetOutputSectionInputSections(ctx)
	// same as frags, need to confirm the containing input sections first
	// so that offset and size can be calculated
	linker.UpdateInputSectionOffsetAndOutputSectionSizeAlign(ctx)

	writers := linker.CollectOutputSectionWritersAndMergedSectionWriters(ctx)
	ctx.OutputWriters = append(ctx.OutputWriters, writers...)
	linker.SortOutputWriters(ctx)

	// size cannot be confirmed until all writers all confirmed
	for _, o := range ctx.OutputWriters {
		o.UpdateSize(ctx)
	}

	// only TLS symbols will appear in GOT
	linker.ScanRelsAndAddSymsToGot(ctx)

	// set offset of all the writers
	// should be after sizes are set
	fileSize := linker.SetOutputShdrOffsets(ctx)
	println("File Size:", fileSize, "bytes")
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
