package linker

import (
	"debug/elf"
	"strings"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

type InputSection struct {
	ObjFile *ObjectFile
	Content []byte
	Shndx uint32
	SecSize uint64
	IsAlive bool
	P2Align uint8
	Name string
	Shdr *Shdr
}

func NewInputSection(obj *ObjectFile, content []byte, shndx uint32, shdr *Shdr, name string) *InputSection {
	return &InputSection {
		ObjFile: obj,
		Content: content,
		Shndx: shndx,
		IsAlive: true,
		P2Align: 1,
		Name: name,
		Shdr: shdr,
	}
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
		(i.Shdr.Flags & uint64(elf.SHF_MERGE)) != 0 {
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