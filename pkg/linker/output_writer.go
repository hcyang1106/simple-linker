package linker

import "debug/elf"

type iOutputWriter interface {
	GetShdr() *Shdr
	CopyBuf(ctx *Context)
	GetName() string
	UpdateSize(ctx *Context)
	GetShndx() int64
}

type OutputWriter struct {
	Name  string
	Shdr  Shdr
	Shndx int64
}

func NewOutputWriter() *OutputWriter {
	return &OutputWriter{
		Shdr: Shdr{
			AddrAlign: 1,
		},
	}
}

func (o *OutputWriter) GetName() string {
	return o.Name
}

func (o *OutputWriter) GetShdr() *Shdr {
	return &o.Shdr
}

func (o *OutputWriter) GetShndx() int64 {
	return o.Shndx
}

func (o *OutputWriter) UpdateSize(ctx *Context) {
	// left empty for successor to implement
}

func (o *OutputWriter) CopyBuf(ctx *Context) {
	// left empty for successor to implement
}

func isTLS(o iOutputWriter) bool {
	return o.GetShdr().Flags&uint64(elf.SHF_TLS) != 0
}

func isBSS(o iOutputWriter) bool {
	return o.GetShdr().Type == uint32(elf.SHT_NOBITS) && !isTLS(o)
}

func isNOTE(o iOutputWriter) bool {
	return o.GetShdr().Type == uint32(elf.SHT_NOTE) &&
		o.GetShdr().Flags&uint64(elf.SHF_ALLOC) != 0
}

func isNONALLOC(o iOutputWriter) bool {
	if o.GetShdr().Flags&uint64(elf.SHF_ALLOC) == 0 {
		return true
	}
	return false
}

func outputWriterAttrToPhdrFlags(o iOutputWriter) uint32 {
	// must be readable
	flags := uint32(elf.PF_R)
	write := o.GetShdr().Flags&uint64(elf.SHF_WRITE) != 0
	if write {
		flags |= uint32(elf.PF_W)
	}
	if o.GetShdr().Flags&uint64(elf.SHF_EXECINSTR) != 0 {
		flags |= uint32(elf.PF_X)
	}
	return flags
}
