package linker

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

// should setup size, offset before using
type OutputEhdrWriter struct {
	OutputWriter
}

func NewOutputEhdrWriter() *OutputEhdrWriter {
	return &OutputEhdrWriter{
		OutputWriter{
			Name: "ehdr",
			Shdr: Shdr{
				Flags:     uint64(elf.SHF_ALLOC),
				Size:      uint64(EhdrSize),
				AddrAlign: 8,
			},
		},
	}
}

func getEntryAddress(ctx *Context) uint64 {
	for _, osec := range ctx.OutputSections {
		if osec.Name == ".text" {
			return osec.Shdr.Addr
		}
	}
	return 0
}

// check if there are compressed instructions or not (32 -> 16)
func getFlags(ctx *Context) uint32 {
	utils.Assert(len(ctx.Args.ObjFiles) > 0)
	flags := ctx.Args.ObjFiles[0].GetEhdr().Flags
	for _, obj := range ctx.Args.ObjFiles[1:] {
		if obj.GetEhdr().Flags&EF_RISCV_RVC != 0 {
			flags |= EF_RISCV_RVC
			break
		}
	}

	return flags
}

func (o *OutputEhdrWriter) CopyBuf(ctx *Context) {
	ehdr := &Ehdr{}
	WriteMagic(ehdr.Ident[:])
	ehdr.Ident[elf.EI_CLASS] = uint8(elf.ELFCLASS64)
	ehdr.Ident[elf.EI_DATA] = uint8(elf.ELFDATA2LSB)
	ehdr.Ident[elf.EI_VERSION] = uint8(elf.EV_CURRENT) // fixed usage
	ehdr.Ident[elf.EI_OSABI] = 0
	ehdr.Ident[elf.EI_ABIVERSION] = 0
	ehdr.Flags = getFlags(ctx)
	ehdr.Type = uint16(elf.ET_EXEC)
	ehdr.Machine = uint16(elf.EM_RISCV)
	ehdr.Version = uint32(elf.EV_CURRENT)
	ehdr.Entry = getEntryAddress(ctx)
	ehdr.EhSize = uint16(EhdrSize)
	ehdr.PhEntSize = uint16(PhdrSize)
	ehdr.ShOff = ctx.OutputShdrsWriter.Shdr.Offset
	ehdr.ShEntSize = uint16(ShdrSize)
	ehdr.PhOff = ctx.OutputPhdrsWriter.Shdr.Offset
	ehdr.PhNum = uint16(ctx.OutputPhdrsWriter.Shdr.Size / uint64(PhdrSize))
	ehdr.ShNum = uint16(ctx.OutputShdrsWriter.Shdr.Size / uint64(ShdrSize))
	buf := bytes.Buffer{}
	err := binary.Write(&buf, binary.LittleEndian, ehdr)
	utils.MustNo(err)
	copy(ctx.Buf[:], buf.Bytes())
}
