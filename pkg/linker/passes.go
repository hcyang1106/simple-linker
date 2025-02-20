package linker

import (
	"debug/elf"
	"github.com/hcyang1106/simple-linker/pkg/utils"
	"math"
	"sort"
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

// change symbol corresponding sections to "fragments"
func ChangeMSecsSymbolsSection(ctx *Context) {
	for _, file := range ctx.Args.ObjFiles {
		file.ChangeMSecsSymbolsSection()
	}
}

func CreateSpecialWriters(ctx *Context) {
	push := func(o iOutputWriter) iOutputWriter {
		ctx.OutputWriters = append(ctx.OutputWriters, o)
		return o
	}
	ctx.OutputEhdrWriter = push(NewOutputEhdrWriter()).(*OutputEhdrWriter)
	ctx.OutputPhdrsWriter = push(NewOutputPhdrsWriter()).(*OutputPhdrsWriter)
	ctx.OutputShdrsWriter = push(NewOutputShdrsWriter()).(*OutputShdrsWriter)
	ctx.OutputGotSectionWriter = push(NewOutputGotSectionWriter()).(*OutputGotSectionWriter)
}

// get called after CreateSpecialWriters,
// since OutputWriters have to be filled
func SetOutputShdrOffsets(ctx *Context) uint64 {
	addr := ADDR_BASE
	for _, o := range ctx.OutputWriters {
		if o.GetShdr().Flags&uint64(elf.SHF_ALLOC) == 0 {
			continue
		}

		addr = utils.AlignTo(addr, o.GetShdr().AddrAlign)
		o.GetShdr().Addr = addr

		if !isTBSS(o) {
			addr += o.GetShdr().Size
		}
	}

	i := 0
	first := ctx.OutputWriters[0]
	for {
		shdr := ctx.OutputWriters[i].GetShdr()
		shdr.Offset = shdr.Addr - first.GetShdr().Addr
		i++

		if i >= len(ctx.OutputWriters) ||
			ctx.OutputWriters[i].GetShdr().Flags&uint64(elf.SHF_ALLOC) == 0 {
			break
		}
	}

	lastShdr := ctx.OutputWriters[i-1].GetShdr()
	fileoff := lastShdr.Offset + lastShdr.Size

	for ; i < len(ctx.OutputWriters); i++ {
		shdr := ctx.OutputWriters[i].GetShdr()
		fileoff = utils.AlignTo(fileoff, shdr.AddrAlign)
		shdr.Offset = fileoff
		fileoff += shdr.Size
	}

	ctx.OutputPhdrsWriter.UpdateSize(ctx)
	return fileoff
}

func UpdateInputSectionOffsetAndOutputSectionSizeAlign(ctx *Context) {
	for _, osec := range ctx.OutputSections {
		offset := uint64(0)
		p2align := int64(0)
		for _, isec := range osec.InputSections {
			offset = utils.AlignTo(offset, 1<<isec.P2Align)
			isec.Offset = uint32(offset)
			offset += uint64(isec.SecSize)
			p2align = int64(math.Max(float64(p2align), float64(isec.P2Align)))
		}
		osec.Shdr.Size = offset
		osec.Shdr.AddrAlign = 1 << p2align
	}
}

func SetOutputSectionInputSections(ctx *Context) {
	for _, file := range ctx.Args.ObjFiles {
		for _, isec := range file.InputSections {
			if isec == nil || !isec.IsAlive {
				continue
			}
			isec.OutputSection.InputSections = append(isec.OutputSection.InputSections, isec)
		}
	}
}

func CollectOutputSectionWritersAndMergedSectionWriters(ctx *Context) []iOutputWriter {
	osecs := make([]iOutputWriter, 0)
	for _, osec := range ctx.OutputSections {
		if len(osec.InputSections) > 0 { // necessary
			osecs = append(osecs, osec)
		}
	}
	for _, osec := range ctx.MergedSections {
		if osec.Shdr.Size > 0 {
			osecs = append(osecs, osec)
		}
	}
	return osecs
}

func SortOutputWriters(ctx *Context) {
	rank := func(o iOutputWriter) int32 {
		typ := o.GetShdr().Type
		flags := o.GetShdr().Flags

		// non-allocs are behind
		if flags&uint64(elf.SHF_ALLOC) == 0 {
			return math.MaxInt32 - 1
		}
		if o == ctx.OutputShdrsWriter {
			return math.MaxInt32
		}
		if o == ctx.OutputPhdrsWriter {
			return 1
		}
		if o == ctx.OutputEhdrWriter {
			return 0
		}
		if typ == uint32(elf.SHT_NOTE) {
			return 2
		}

		toBit := func(b bool) int {
			if b {
				return 1
			}
			return 0
		}

		// non-writable first
		writeable := toBit(flags&uint64(elf.SHF_WRITE) != 0)
		notExec := toBit(flags&uint64(elf.SHF_EXECINSTR) == 0)
		notTls := toBit(flags&uint64(elf.SHF_TLS) == 0)
		isBss := toBit(typ == uint32(elf.SHT_NOBITS))

		return int32(writeable<<7 | notExec<<6 | notTls<<5 | isBss<<4)
	}

	// same values' order remain the same
	// smaller rank first
	sort.SliceStable(ctx.OutputWriters, func(i, j int) bool {
		return rank(ctx.OutputWriters[i]) < rank(ctx.OutputWriters[j])
	})
}

func UpdateFragmentOffsetAndMergedSectionSizeAlign(ctx *Context) {
	for _, m := range ctx.MergedSections {
		m.AssignFragmentsOffsets()
	}
}

func ScanRelsAndAddSymsToGot(ctx *Context) {
	for _, file := range ctx.Args.ObjFiles {
		file.ScanRelsFindGotSyms()
	}
	for _, file := range ctx.Args.ObjFiles {
		for _, sym := range file.Symbols {
			if sym.File == file && (sym.Flags&IsInGot) != 0 {
				ctx.OutputGotSectionWriter.AddGotTLSSym(sym)
			}
		}
	}
}
