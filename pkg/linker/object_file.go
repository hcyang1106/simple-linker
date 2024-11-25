package linker

import (
	"debug/elf"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

type ObjectFile struct {
	File           *File
	ElfEhdr        Ehdr
	ElfSecHdrs     []Shdr
	ElfSyms        []Sym
	SymTabSecHdr   *Shdr // already saved in ElfSecHdrs, so use pointer here
	FirstGlobal    uint32
	ShStrTab       []byte
	SymStrTab      []byte
	SymtabShndxSec []uint32

	IsAlive       bool // active or not (inactive indicates in lib), which means finding symbols needed
	InputSections []*InputSection
	Symbols       []*Symbol
	LocalSymbols  []*Symbol
	TotalSyms     uint32
	TotalSecs     uint32

}

// fill in ElfEhdr, ElfSecHdrs, ShStrTab
func NewObjectFile(file *File, isAlive bool, ctx *Context) {
	f := ObjectFile{
		File:       file,
		ElfSecHdrs: []Shdr{},
		IsAlive:    isAlive,
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

	numSecs := (uint32)(f.ElfEhdr.ShNum)
	if numSecs == 0 {
		numSecs = (uint32)(f.ElfSecHdrs[0].Size)
	}
	f.TotalSecs = numSecs

	var i uint32
	for i = 0; i < numSecs-1; i++ {
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

	f.Parse(ctx)
	ctx.Args.ObjFiles = append(ctx.Args.ObjFiles, &f)
}

func (f *ObjectFile) GetBytesFromShdr(s *Shdr) []byte {
	end := s.Offset + s.Size
	if end > uint64(len(f.File.Content)) {
		utils.Fatal("Get bytes exceeds file length")
	}

	return f.File.Content[s.Offset:end]
}

func (f *ObjectFile) GetBytesFromIdx(idx uint32) []byte {
	if idx > uint32(len(f.ElfSecHdrs)) {
		utils.Fatal("Read index exceeds section header table length")
	}

	shdr := &f.ElfSecHdrs[idx]
	return f.GetBytesFromShdr(shdr)
}

func (f *ObjectFile) FindSectionHdr(secType uint32) *Shdr {
	for _, shdr := range f.ElfSecHdrs {
		if shdr.Type == secType {
			return &shdr
		}
	}
	return nil
}

func (f *ObjectFile) FillInElfSyms(shdr *Shdr) {
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
func (f *ObjectFile) ParseSymTab() {
	f.SymTabSecHdr = f.FindSectionHdr(uint32(elf.SHT_SYMTAB))
	if f.SymTabSecHdr != nil {
		f.FirstGlobal = f.SymTabSecHdr.Info
		f.FillInElfSyms(f.SymTabSecHdr)
		f.SymStrTab = f.GetBytesFromIdx(f.SymTabSecHdr.Link)
	}
}

func (f *ObjectFile) ParseSymtabShndxSec() {
	secHdr := f.FindSectionHdr(uint32(elf.SHT_SYMTAB_SHNDX))
	if secHdr != nil {
		content := f.GetBytesFromShdr(secHdr)
		f.SymtabShndxSec = utils.ReadSlice[uint32](content, 4)
	}
}

// fill in LocalSymbols and Symbols field
// two kinds of special symbols, abs and undefined
// abs => no section
// special sections' symbol => input sections is not filled
func (f *ObjectFile) ParseSymbols(ctx *Context) {
	f.LocalSymbols = make([]*Symbol, 0)
	f.Symbols = make([]*Symbol, 0)

	var i uint32
	for _, esym := range f.ElfSyms {
		if i == 0 {
			// first symbol is not used
			first := NewSymbol(f, "")
			f.LocalSymbols = append(f.LocalSymbols, first)
			f.Symbols = append(f.Symbols, first)
			i += 1
			continue
		}

		name := ElfGetName(f.SymStrTab, esym.Name)
		sym := NewSymbol(f, name)
		sym.SetValue(esym.Val)
		sym.SetSymIdx(i)
		if !esym.IsAbs() {
			shndx := esym.GetShndx(f.SymtabShndxSec, i)
			sym.SetInputSection(f.InputSections[shndx])
		}

		if i < f.FirstGlobal {
			f.LocalSymbols = append(f.LocalSymbols, sym)
			f.Symbols = append(f.Symbols, sym)
			i += 1
			continue
		}
		gSym := ctx.GetSymbol(name)
		f.Symbols = append(f.Symbols, gSym)
		if !esym.IsUndef() {
			*gSym = *sym
		}
		i += 1
	}

	f.TotalSyms = i
}

func (f *ObjectFile) Parse(ctx *Context) {
	f.ParseSymTab()
	f.ParseSymtabShndxSec()
	f.ParseInputSections()
	f.ParseSymbols(ctx) // should be after parsing sections
}

// fill in input sections field
func (f *ObjectFile) ParseInputSections() {
	var i uint32
	for _, hdr := range f.ElfSecHdrs {
		switch elf.SectionType(hdr.Type) {
		// ignore these sections
		//case elf.SHT_GROUP, elf.SHT_SYMTAB, elf.SHT_STRTAB, elf.SHT_REL,
		//	elf.SHT_RELA, elf.SHT_NULL, elf.SHT_SYMTAB_SHNDX:
		//	iSection := NewInputSection(f, nil, i)
		//	f.InputSections = append(f.InputSections, iSection)
		//	break
		default:
			iContent := f.GetBytesFromIdx(i)
			iSection := NewInputSection(f, iContent, i)
			f.InputSections = append(f.InputSections, iSection)
		}
		i += 1
	}
}

func (f *ObjectFile) MarkLiveObjects(ctx *Context, roots []*ObjectFile) []*ObjectFile {
	for i := f.FirstGlobal; i < f.TotalSyms; i++ {
		esym := f.ElfSyms[i]
		sym := f.Symbols[i]
		//necessary
		if sym.File == nil {
			continue
		}
		if esym.IsUndef() && !sym.File.IsAlive {
			sym.File.IsAlive = true
			roots = append(roots, sym.File)
		}
	}
	return roots
}

func (f *ObjectFile) ClearUnusedGlobalSymbols(ctx *Context) {
	var i uint32
	for i = f.FirstGlobal; i < f.TotalSyms; i++ {
		sym := f.Symbols[i]
		delete(ctx.SymbolMap, sym.Name)
	}
}
