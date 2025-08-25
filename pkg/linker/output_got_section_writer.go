package linker

import (
	"debug/elf"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

type OutputGotSectionWriter struct {
	OutputWriter
	GotTLSSyms []*Symbol
}

func NewOutputGotSectionWriter() *OutputGotSectionWriter {
	g := &OutputGotSectionWriter{OutputWriter: *NewOutputWriter()}
	g.Name = ".got"
	g.Shdr.Type = uint32(elf.SHT_PROGBITS)
	g.Shdr.Flags = uint64(elf.SHF_ALLOC | elf.SHF_WRITE)
	return g
}

func (g *OutputGotSectionWriter) AddGotTLSSym(sym *Symbol) {
	sym.GotEntryIdx = uint32(len(g.GotTLSSyms))
	g.GotTLSSyms = append(g.GotTLSSyms, sym)
	g.Shdr.Size += 8
}

func (g *OutputGotSectionWriter) CopyBuf(ctx *Context) {
	base := ctx.Buf[g.Shdr.Offset:]
	for idx, sym := range g.GotTLSSyms {
		utils.Write[uint64](base[idx*8:], sym.GetAddr() - ctx.TLSSegmentAddr) // stores the offset from tp!
	}
}
