package linker

import "debug/elf"

type OutputSection struct {
	OutputWriter
	InputSections []*InputSection
	Idx           uint32 // the index in ctx.OutputSections
}

func NewOutputSection(
	name string, typ uint32, flags uint64, idx uint32) *OutputSection {
	o := &OutputSection{OutputWriter: *NewOutputWriter()}
	o.Name = name
	o.Shdr.Type = typ
	o.Shdr.Flags = flags
	o.Idx = idx
	return o
}

func (o *OutputSection) CopyBuf(ctx *Context) {
	if o.Shdr.Type == uint32(elf.SHT_NOBITS) {
		return
	}

	base := ctx.Buf[o.Shdr.Offset:]
	for _, isec := range o.InputSections {
		isec.WriteTo(ctx, base[isec.Offset:])
	}
}

// check if is thread bss section
func isTBSS(o iOutputWriter) bool {
	shdr := o.GetShdr()
	return shdr.Type == uint32(elf.SHT_NOBITS) &&
		shdr.Flags&uint64(elf.SHF_TLS) != 0
}
