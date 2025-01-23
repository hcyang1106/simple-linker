package linker

import (
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

func MarkLiveObjects(ctx *Context) {
	roots := make([]*ObjectFile, 0)
	for _, file := range ctx.Args.ObjFiles {
		if file.IsAlive {
			roots = append(roots, file)
		}
	}
	for len(roots) > 0 {
		roots = roots[0].MarkLiveObjects(ctx, roots)
		roots = roots[1:]
	}
}

func ClearSymbolsAndFiles(ctx *Context) {
	ClearUnusedGlobalSymbols(ctx)
	ClearUnusedFiles(ctx)
}

func ClearUnusedGlobalSymbols(ctx *Context) {
	for _, file := range ctx.Args.ObjFiles {
		if !file.IsAlive {
			file.ClearUnusedGlobalSymbols(ctx)
		}
	}
}

func ClearUnusedFiles(ctx *Context) {
	var i int = 0
	for _, file := range ctx.Args.ObjFiles {
		if file.IsAlive {
			ctx.Args.ObjFiles[i] = file
			i += 1
		}
	}
	ctx.Args.ObjFiles = ctx.Args.ObjFiles[:i]
}

// change symbol corresponding sections to "segments"
func ChangeMSecsSymbolsSection(ctx *Context) {
	for _, file := range ctx.Args.ObjFiles {
		file.ChangeMSecsSymbolsSection()
	}
}

func CreateSyntheticSections(ctx *Context) {
	ctx.OutputEhdrWriter = NewOutputEhdrWriter()
	ctx.OutputWriters = append(ctx.OutputWriters, ctx.OutputEhdrWriter)
}

// get called after CreateSyntheticSections,
// since OutputWriters have to be filled
func GetFileSize(ctx *Context) uint64 {
	offset := uint64(0)
	for _, o := range ctx.OutputWriters {
		offset = utils.AlignTo(offset, o.GetShdr().AddrAlign)
		offset += o.GetShdr().Size
	}
	return offset
}