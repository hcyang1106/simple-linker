package linker

import (
	"debug/elf"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

type InputFile struct {
	File *File
	ElfEhdr Ehdr
	ElfSecHdrs []Shdr
	ElfSyms []Sym
	SymTabSecHdr *Shdr // already saved in ElfSecHdrs, so use pointer here
	FirstGlobal uint32
	ShStrTab []byte
	SymStrTab []byte
}

func NewInputFile(file *File) *InputFile {
	f := InputFile{
		File: file,
		ElfSecHdrs: []Shdr{},
	}

	if len(file.Content) < EhdrSize {
		utils.Fatal("file is smaller than Ehdr size")
	}
	MustHaveMagic(file.Content)

	utils.Read[Ehdr](file.Content, &f.ElfEhdr)

	secHdrContent := file.Content[f.ElfEhdr.ShOff:]
	shdr := Shdr{}
	utils.Read[Shdr](secHdrContent, &shdr)
	f.ElfSecHdrs = append(f.ElfSecHdrs, shdr)

	numSecs := (int)(f.ElfEhdr.ShNum)
	if (numSecs == 0) {
		numSecs = (int)(f.ElfSecHdrs[0].Size)
	}

	for i := 0; i < numSecs - 1; i++ {
		secHdrContent = secHdrContent[ShdrSize:]
		shdr = Shdr{}
		utils.Read[Shdr](secHdrContent, &shdr)
		f.ElfSecHdrs = append(f.ElfSecHdrs, shdr)
	}

	shStrndx := uint32(f.ElfEhdr.ShStrndx)
	if shStrndx == uint32(elf.SHN_XINDEX) {
		shStrndx = f.ElfSecHdrs[0].Link
	}
	f.ShStrTab = f.GetBytesFromIdx(shStrndx)

	return &f
}

func (f *InputFile) GetBytesFromShdr(s *Shdr) []byte {
	end := s.Offset + s.Size
	if end > uint64(len(f.File.Content)) {
		utils.Fatal("Get bytes exceeds file length")
	}

	return f.File.Content[s.Offset:end]
}

func (f *InputFile) GetBytesFromIdx(idx uint32) []byte {
	if idx > uint32(len(f.ElfSecHdrs)) {
		utils.Fatal("Read index exceeds section header table length")
	}

	shdr := &f.ElfSecHdrs[idx]
	return f.GetBytesFromShdr(shdr)
}

func (f *InputFile) FindSectionHdr(secType uint32) *Shdr {
	for _, shdr := range f.ElfSecHdrs {
		if shdr.Type == secType {
			return &shdr
		}
	}

	return nil
}

func (f *InputFile) FillInElfSyms(shdr *Shdr) {
	bs := f.GetBytesFromShdr(shdr)
	nums := len(bs) / SymSize
	f.ElfSyms = make([]Sym, nums)
	for i := 0; i < nums; i++ {
		s := Sym{}
		utils.Read[Sym](bs, &s)
		f.ElfSyms[i] = s
		bs = bs[SymSize:] // does not panic if idx reaches length
	}
}

// find symbol table section header and
// create symbol array
func (f *InputFile) ParseSymTab() {
	f.SymTabSecHdr = f.FindSectionHdr(uint32(elf.SHT_SYMTAB))
	if (f.SymTabSecHdr != nil) {
		f.FirstGlobal = f.SymTabSecHdr.Info
		f.FillInElfSyms(f.SymTabSecHdr)
		f.SymStrTab = f.GetBytesFromIdx(f.SymTabSecHdr.Link)
	}
}

