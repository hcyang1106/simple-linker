package linker

import (
	"bytes"
	"debug/elf"
	"strconv"
	"strings"
	"unsafe"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

const EhdrSize = int(unsafe.Sizeof(Ehdr{}))
const ShdrSize = int(unsafe.Sizeof(Shdr{}))
const SymSize = int(unsafe.Sizeof(Sym{}))
const PhdrSize = int(unsafe.Sizeof(Phdr{}))
const AhdrSize = int(unsafe.Sizeof(ArHdr{}))

type Ehdr struct {
	Ident     [16]uint8
	Type      uint16
	Machine   uint16
	Version   uint32
	Entry     uint64
	PhOff     uint64
	ShOff     uint64
	Flags     uint32
	EhSize    uint16
	PhEntSize uint16
	PhNum     uint16
	ShEntSize uint16
	ShNum     uint16
	ShStrndx  uint16
}

type Shdr struct {
	Name      uint32
	Type      uint32
	Flags     uint64
	Addr      uint64
	Offset    uint64
	Size      uint64
	Link      uint32
	Info      uint32
	AddrAlign uint64
	EntSize   uint64
}

type Phdr struct {
	Type     uint32
	Flags    uint32
	Offset   uint64
	VAddr    uint64
	PAddr    uint64
	FileSize uint64
	MemSize  uint64
	Align    uint64
}

type Sym struct {
	Name  uint32
	Info  uint8
	Other uint8
	Shndx uint16
	Val   uint64
	Size  uint64
}

func (s *Sym) GetShndx(table []uint32, idx uint32) uint32 {
	if elf.SectionIndex(s.Shndx) != elf.SHN_XINDEX {
		return uint32(s.Shndx)
	}
	return table[idx]
}

func (s *Sym) IsAbs() bool {
	return s.Shndx == uint16(elf.SHN_ABS)
}

func (s *Sym) IsUndef() bool {
	return s.Shndx == uint16(elf.SHN_UNDEF)
}

type ArHdr struct {
	Name [16]byte
	Date [12]byte
	Uid  [6]byte
	Gid  [6]byte
	Mode [8]byte
	Size [10]byte
	Fmag [2]byte
}

func (a *ArHdr) HasPrefix(s string) bool {
	return strings.HasPrefix(string(a.Name[:]), s)
}

func (a *ArHdr) IsStrTab() bool {
	return a.HasPrefix("// ")
}

func (a *ArHdr) IsSymtab() bool {
	return a.HasPrefix("/ ") || a.HasPrefix("/SYM64/ ")
}

func (a *ArHdr) GetSize() int {
	trimmed := strings.TrimSpace(string(a.Size[:]))
	size, err := strconv.Atoi(trimmed)
	utils.MustNo(err)
	return size
}

func (a *ArHdr)  ReadName(strTab []byte) string {
	// Long Name
	// "/123    " => the number is the start index in strTab
	if a.HasPrefix("/") {
		trimmed := strings.TrimSpace(string(a.Name[1:]))
		start, err := strconv.Atoi(trimmed)
		utils.MustNo(err)
		end := start + bytes.Index(strTab[start:], []byte("/\n"))
		return string(strTab[start:end])
	}
	// Short Name
	end := bytes.Index(a.Name[:], []byte("/"))
	utils.Assert(end != -1)
	return string(a.Name[:end])
}

func ElfGetName(strTab []byte, offset uint32) string {
	length := uint32(bytes.Index(strTab[offset:], []byte{0}))
	return string(strTab[offset:offset + length])
}
