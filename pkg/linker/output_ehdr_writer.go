package linker

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

type OutputEhdrWriter struct {
	OutputWriter
}

func NewOutputEhdrWriter() *OutputEhdrWriter {
	return &OutputEhdrWriter{
		OutputWriter{
			Shdr: Shdr{
				Flags:     uint64(elf.SHF_ALLOC),
				Size:      uint64(EhdrSize),
				AddrAlign: 8,
			},
		},
	}
}

func (o *OutputEhdrWriter) CopyBuf(ctx *Context) {
	ehdr := &Ehdr{}
	WriteMagic(ehdr.Ident[:])
	ehdr.Ident[elf.EI_CLASS] = uint8(elf.ELFCLASS64)
	ehdr.Ident[elf.EI_DATA] = uint8(elf.ELFDATA2LSB)
	ehdr.Ident[elf.EI_VERSION] = uint8(elf.EV_CURRENT) // fixed usage
	ehdr.Ident[elf.EI_OSABI] = 0
	ehdr.Ident[elf.EI_ABIVERSION] = 0

	ehdr.Type = uint16(elf.ET_EXEC)
	ehdr.Machine = uint16(elf.EM_RISCV)
	ehdr.Version = uint32(elf.EV_CURRENT)
	// TODO
	ehdr.EhSize = uint16(EhdrSize)
	ehdr.PhEntSize = uint16(PhdrSize)
	// TODO
	ehdr.ShEntSize = uint16(ShdrSize)

	buf := bytes.Buffer{}
	err := binary.Write(&buf, binary.LittleEndian, ehdr)
	utils.MustNo(err)
	copy(ctx.Buf[:], buf.Bytes())
}
