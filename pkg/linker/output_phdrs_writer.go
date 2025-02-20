package linker

import (
	"debug/elf"
	"math"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

type OutputPhdrsWriter struct {
	OutputWriter
	Phdrs []Phdr
}

func NewOutputPhdrsWriter() *OutputPhdrsWriter {
	return &OutputPhdrsWriter{
		OutputWriter: OutputWriter{
			Name: "phdr",
			Shdr: Shdr{
				AddrAlign: 8,
				Flags:     uint64(elf.SHF_ALLOC),
			},
		},
	}
}

func (o *OutputPhdrsWriter) UpdateSize(ctx *Context) {
	o.createPhdrs(ctx)
	o.Shdr.Size = uint64(len(o.Phdrs)) * uint64(PhdrSize)
}

func (o *OutputPhdrsWriter) CopyBuf(ctx *Context) {
	start := ctx.Buf[o.Shdr.Offset:]
	for _, phdr := range ctx.OutputPhdrsWriter.Phdrs {
		utils.Write[Phdr](start, phdr)
		start = start[PhdrSize:]
	}
}

// phdr file size represents the size took in the file
// phdr memory size represents the size occupying the mem
// mem size >= file size because of bss
// phdrs: phdr, note
func (o *OutputPhdrsWriter) createPhdrs(ctx *Context) {
	o.Phdrs = make([]Phdr, 0)
	define := func(typ, flags uint32, minAlign uint64, outputWriter iOutputWriter) {
		o.Phdrs = append(o.Phdrs, Phdr{})
		phdr := &o.Phdrs[len(o.Phdrs)-1]
		phdr.Type = typ
		phdr.Flags = flags
		phdr.Align = uint64(math.Max(
			float64(minAlign),
			float64(outputWriter.GetShdr().AddrAlign)))
		phdr.Offset = outputWriter.GetShdr().Offset
		if outputWriter.GetShdr().Type == uint32(elf.SHT_NOBITS) {
			phdr.FileSize = 0
		} else {
			phdr.FileSize = outputWriter.GetShdr().Size
		}
		phdr.VAddr = outputWriter.GetShdr().Addr
		phdr.PAddr = outputWriter.GetShdr().Addr
		phdr.MemSize = outputWriter.GetShdr().Size
	}

	// size = arriving outputwriter end address - phdr start address
	push := func(outputWriter iOutputWriter) {
		phdr := &o.Phdrs[len(o.Phdrs)-1]
		phdr.Align = uint64(math.Max(
			float64(phdr.Align),
			float64(outputWriter.GetShdr().AddrAlign)))
		if outputWriter.GetShdr().Type != uint32(elf.SHT_NOBITS) {
			phdr.FileSize = outputWriter.GetShdr().Addr +
				outputWriter.GetShdr().Size -
				phdr.VAddr
		}
		phdr.MemSize = outputWriter.GetShdr().Addr +
			outputWriter.GetShdr().Size -
			phdr.VAddr
	}

	// phdr segment
	define(uint32(elf.PT_PHDR), uint32(elf.PF_R), 8, ctx.OutputPhdrsWriter)

	// note segment
	for i := 0; i < len(ctx.OutputWriters); i++ {
		iCurr := ctx.OutputWriters[i]
		if !isNOTE(iCurr) {
			continue
		}
		flags := outputWriterAttrToPhdrFlags(iCurr)
		align := iCurr.GetShdr().AddrAlign
		define(uint32(elf.PT_NOTE), flags, align, iCurr)
		for j := i + 1; j < len(ctx.OutputWriters); j++ {
			jCurr := ctx.OutputWriters[j]
			if !isNOTE(jCurr) || outputWriterAttrToPhdrFlags(jCurr) != flags {
				break
			}
			push(jCurr)
		}
	}

	// load segment
	outputWriters := make([]iOutputWriter, 0)
	for _, o := range ctx.OutputWriters {
		outputWriters = append(outputWriters, o)
	}

	var i int
	for _, outputWriter := range outputWriters {
		if isTBSS(outputWriter) {
			continue
		}
		outputWriters[i] = outputWriter
		i++
	}
	outputWriters = outputWriters[:i]

	for i := 0; i < len(outputWriters); {
		curr := outputWriters[i]
		if isNONALLOC(curr) {
			break
		}
		currFlags := outputWriterAttrToPhdrFlags(curr)
		define(uint32(elf.PT_LOAD), currFlags, PageSize, curr)
		i++
		for i < len(outputWriters) && !isBSS(outputWriters[i]) &&
			outputWriterAttrToPhdrFlags(outputWriters[i]) == currFlags {
			push(outputWriters[i])
			i++
		}
		for i < len(outputWriters) && isBSS(outputWriters[i]) &&
			outputWriterAttrToPhdrFlags(outputWriters[i]) == currFlags {
			push(outputWriters[i])
			i++
		}
	}

	// tls segment
	for i := 0; i < len(ctx.OutputWriters); i++ {
		curr := ctx.OutputWriters[i]
		if !isTLS(curr) {
			continue
		}

		currFlags := outputWriterAttrToPhdrFlags(ctx.OutputWriters[i])
		define(uint32(elf.PT_TLS), currFlags, 1, ctx.OutputWriters[i])
		i++
		for i < len(ctx.OutputWriters) && isTLS(ctx.OutputWriters[i]) {
			push(ctx.OutputWriters[i])
			i++
		}

		phdr := &o.Phdrs[len(o.Phdrs)-1]
		ctx.TLSSegmentAddr = phdr.VAddr
	}
}
