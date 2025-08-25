package linker

import (
	"debug/elf"
	"github.com/hcyang1106/simple-linker/pkg/utils"
	"strings"
)

type InputSection struct {
	ObjFile       *ObjectFile
	Content       []byte
	Shndx         uint32
	SecSize       uint64
	IsAlive       bool
	P2Align       uint8
	Name          string
	Shdr          *Shdr
	OutputSection *OutputSection
	Offset        uint32 // the offset inside output section
	RelSecIdx     uint32 // corresponding relocation section
	Rels          []Rela
}

func NewInputSection(obj *ObjectFile, content []byte, shndx uint32, shdr *Shdr, name string) *InputSection {
	return &InputSection{
		ObjFile: obj,
		Content: content,
		Shndx:   shndx,
		IsAlive: true,
		P2Align: 1,
		Name:    name,
		Shdr:    shdr,
	}
}

func (i *InputSection) GetInputSectionOutputSection(ctx *Context) *OutputSection {
	oName := i.GetOutputSectionName()
	oFlags := i.Shdr.Flags &^ uint64(elf.SHF_GROUP) &^
		uint64(elf.SHF_COMPRESSED) &^ uint64(elf.SHF_LINK_ORDER) // remove these flags

	find := func() *OutputSection {
		for _, osec := range ctx.OutputSections {
			if oName == osec.Name && i.Shdr.Type == osec.Shdr.Type &&
				oFlags == osec.Shdr.Flags {
				return osec
			}
		}
		return nil
	}

	if osec := find(); osec != nil {
		return osec
	}

	// nil, create one
	osec := NewOutputSection(oName, i.Shdr.Type, oFlags, uint32(len(ctx.OutputSections)))
	ctx.OutputSections = append(ctx.OutputSections, osec)

	return osec
}

func (i *InputSection) SetInputSectionOutputSection(outputSection *OutputSection) {
	i.OutputSection = outputSection
}

func (i *InputSection) SetInputSectionSize(size uint64) {
	i.SecSize = size
}

func (i *InputSection) SetP2Align(align uint64) {
	if align == 0 {
		return
	}
	i.P2Align = utils.CountZeros(align)
}

func (i *InputSection) GetOutputSectionName() string {
	// only mergeable rodata
	if (i.Name == ".rodata" || strings.HasPrefix(i.Name, ".rodata")) &&
		(i.Shdr.Flags&uint64(elf.SHF_MERGE)) != 0 {
		if (i.Shdr.Flags & uint64(elf.SHF_STRINGS)) != 0 {
			return ".rodata.str"
		}
		return ".rodata.cst"
	}

	var prefixes = []string{
		".text", ".data.rel.ro", ".data", ".rodata", ".bss.rel.ro", ".bss",
		".init_array", ".fini_array", ".tbss", ".tdata", ".gcc_except_table",
		".ctors", ".dtors",
	}
	for _, prefix := range prefixes {
		if i.Name == prefix || strings.HasPrefix(i.Name, prefix) {
			return prefix
		}
	}
	return i.Name
}

// this is used to copy the content part, not the header part
// note that bss content is not copied
func (i *InputSection) WriteTo(ctx *Context, buf []byte) {
	if i.Shdr.Type == uint32(elf.SHT_NOBITS) || i.SecSize == 0 {
		return
	}
	copy(buf, i.Content)

	if i.Shdr.Flags&uint64(elf.SHF_ALLOC) != 0 {
		i.ApplyRelocAlloc(ctx, buf)
	}
}

func (i *InputSection) GetRels() []Rela {
	if i.RelSecIdx == 0 {
		return nil
	}
	// already parsed
	if i.Rels != nil {
		return i.Rels
	}
	contents := i.ObjFile.GetBytesFromShdr(&i.ObjFile.ElfSecHdrs[i.RelSecIdx])
	i.Rels = utils.ReadSlice[Rela](contents, RelaSize)
	return i.Rels
}

func (i *InputSection) GetAddr() uint64 {
	return i.OutputSection.Shdr.Addr + uint64(i.Offset)
}

func (i *InputSection) ScanRelsFindGotSyms() {
	for _, rel := range i.GetRels() {
		sym := i.ObjFile.Symbols[rel.Sym]
		if rel.Type == uint32(elf.R_RISCV_TLS_GOT_HI20) {
			sym.Flags |= IsInGot
		}
	}
}

// relocations can be divided into multiple kinds,
// absolute address, pc relative address, or got entry relative address
// base is the starting address of the section
func (i *InputSection) ApplyRelocAlloc(ctx *Context, base []byte) {
	rels := i.GetRels()

	for a := 0; a < len(rels); a++ {
		rel := rels[a]
		if rel.Type == uint32(elf.R_RISCV_NONE) ||
			rel.Type == uint32(elf.R_RISCV_RELAX) {
			continue
		}

		sym := i.ObjFile.Symbols[rel.Sym]
		loc := base[rel.Offset:]

		if sym.File == nil {
			continue
		}

		S := sym.GetAddr()
		A := uint64(rel.Addend)
		P := i.GetAddr() + rel.Offset

		switch elf.R_RISCV(rel.Type) {
		case elf.R_RISCV_32:
			utils.Write[uint32](loc, uint32(S+A))
		case elf.R_RISCV_64:
			utils.Write[uint64](loc, S+A)
		case elf.R_RISCV_BRANCH:
			// pc relative offset
			writeBtype(loc, uint32(S+A-P))
		case elf.R_RISCV_JAL:
			// pc relative offset
			writeJtype(loc, uint32(S+A-P))
		case elf.R_RISCV_CALL, elf.R_RISCV_CALL_PLT:
			// call uses auipc and jalr to jump to a function
			// needs a register as a base so use jalr instead of jal
			// R_RISCV_CALL is now deprecated, R_RISCV_CALL_PLT only
			val := uint32(S + A - P)
			writeUtype(loc, val)
			writeItype(loc[4:], val)
		case elf.R_RISCV_TLS_GOT_HI20:
			utils.Write[uint32](loc, uint32(sym.GetGotEntryAddr(ctx)+A-P))
		case elf.R_RISCV_PCREL_HI20:
			utils.Write[uint32](loc, uint32(S+A-P))
		case elf.R_RISCV_HI20: // %high(symbol)
			writeUtype(loc, uint32(S+A))
		case elf.R_RISCV_LO12_I, elf.R_RISCV_LO12_S:
			val := S + A
			if rel.Type == uint32(elf.R_RISCV_LO12_I) {
				writeItype(loc, uint32(val))
			} else {
				writeStype(loc, uint32(val))
			}

			if utils.SignExtend(val, 11) == val {
				setRs1(loc, 0)
			}
		case elf.R_RISCV_TPREL_LO12_I, elf.R_RISCV_TPREL_LO12_S:
			val := S + A - ctx.TLSSegmentAddr
			if rel.Type == uint32(elf.R_RISCV_TPREL_LO12_I) {
				writeItype(loc, uint32(val))
			} else {
				writeStype(loc, uint32(val))
			}

			if utils.SignExtend(val, 11) == val {
				setRs1(loc, 4)
			}
		}
	}

	// usually combined with R_RISCV_PCREL_HI20 (auipc)
	// %pc_rel(symbol)
	for a := 0; a < len(rels); a++ {
		switch elf.R_RISCV(rels[a].Type) {
		case elf.R_RISCV_PCREL_LO12_I, elf.R_RISCV_PCREL_LO12_S:
			sym := i.ObjFile.Symbols[rels[a].Sym]
			utils.Assert(sym.InputSection == i)
			loc := base[rels[a].Offset:]
			val := utils.ReadWithReturn[uint32](base[sym.Value:])

			if rels[a].Type == uint32(elf.R_RISCV_PCREL_LO12_I) {
				writeItype(loc, val)
			} else {
				writeStype(loc, val)
			}
		}
	}

	for a := 0; a < len(rels); a++ {
		switch elf.R_RISCV(rels[a].Type) {
		case elf.R_RISCV_PCREL_HI20, elf.R_RISCV_TLS_GOT_HI20:
			loc := base[rels[a].Offset:]
			val := utils.ReadWithReturn[uint32](loc)
			utils.Write[uint32](loc, utils.ReadWithReturn[uint32](i.Content[rels[a].Offset:]))
			writeUtype(loc, val)
		}
	}
}

func itype(val uint32) uint32 {
	return val << 20
}

func stype(val uint32) uint32 {
	return utils.Bits(val, 11, 5)<<25 | utils.Bits(val, 4, 0)<<7
}

func btype(val uint32) uint32 {
	return utils.Bit(val, 12)<<31 | utils.Bits(val, 10, 5)<<25 |
		utils.Bits(val, 4, 1)<<8 | utils.Bit(val, 11)<<7
}

func utype(val uint32) uint32 {
	return (val + 0x800) & 0xffff_f000
}

func jtype(val uint32) uint32 {
	return utils.Bit(val, 20)<<31 | utils.Bits(val, 10, 1)<<21 |
		utils.Bit(val, 11)<<20 | utils.Bits(val, 19, 12)<<12
}

func cbtype(val uint16) uint16 {
	return utils.Bit(val, 8)<<12 | utils.Bit(val, 4)<<11 | utils.Bit(val, 3)<<10 |
		utils.Bit(val, 7)<<6 | utils.Bit(val, 6)<<5 | utils.Bit(val, 2)<<4 |
		utils.Bit(val, 1)<<3 | utils.Bit(val, 5)<<2
}

func cjtype(val uint16) uint16 {
	return utils.Bit(val, 11)<<12 | utils.Bit(val, 4)<<11 | utils.Bit(val, 9)<<10 |
		utils.Bit(val, 8)<<9 | utils.Bit(val, 10)<<8 | utils.Bit(val, 6)<<7 |
		utils.Bit(val, 7)<<6 | utils.Bit(val, 3)<<5 | utils.Bit(val, 2)<<4 |
		utils.Bit(val, 1)<<3 | utils.Bit(val, 5)<<2
}

func writeItype(loc []byte, val uint32) {
	mask := uint32(0b000000_00000_11111_111_11111_1111111)
	utils.Write[uint32](loc, (utils.ReadWithReturn[uint32](loc)&mask)|itype(val))
}

func writeStype(loc []byte, val uint32) {
	mask := uint32(0b000000_11111_11111_111_00000_1111111)
	utils.Write[uint32](loc, (utils.ReadWithReturn[uint32](loc)&mask)|stype(val))
}

func writeBtype(loc []byte, val uint32) {
	mask := uint32(0b000000_11111_11111_111_00000_1111111)
	utils.Write[uint32](loc, (utils.ReadWithReturn[uint32](loc)&mask)|btype(val))
}

func writeUtype(loc []byte, val uint32) {
	mask := uint32(0b000000_00000_00000_000_11111_1111111)
	utils.Write[uint32](loc, (utils.ReadWithReturn[uint32](loc)&mask)|utype(val))
}

func writeJtype(loc []byte, val uint32) {
	mask := uint32(0b000000_00000_00000_000_11111_1111111)
	utils.Write[uint32](loc, (utils.ReadWithReturn[uint32](loc)&mask)|jtype(val))
}

func setRs1(loc []byte, rs1 uint32) {
	utils.Write[uint32](loc, utils.ReadWithReturn[uint32](loc)&0b111111_11111_00000_111_11111_1111111)
	utils.Write[uint32](loc, utils.ReadWithReturn[uint32](loc)|(rs1<<15))
}
