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

	MergeableSections []*MergeableSection
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

// the input is symbol table section header
// "Sym" structure is exactly how symbol is stored in elf
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
// symbol table can be found using its type
// however section name string table cannot, so it's stored in Ehdr
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
	f.Symbols = make([]*Symbol, 0) // all symbols

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
		// only global symbols could be undefined
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
	f.ParseSymTab()         // fill in elfSyms (name is simply the offset)
	f.ParseSymtabShndxSec() // if there exist the section
	f.ParseInputSections()
	f.ParseSymbols(ctx)           // should be after parsing sections, set up sym arrays and global syms
	f.ParseMergeableSections(ctx) // create mergeable section array, and store fragments into merged section in ctx
}

// fill in input sections field
// in demonstrated code, if section type falls in those special types,
// then fill in nil in the array
func (f *ObjectFile) ParseInputSections() {
	var i uint32
	for _, hdr := range f.ElfSecHdrs {
		switch elf.SectionType(hdr.Type) {
		// ignore these sections
		case elf.SHT_GROUP, elf.SHT_SYMTAB, elf.SHT_STRTAB, elf.SHT_REL,
			elf.SHT_RELA, elf.SHT_NULL, elf.SHT_SYMTAB_SHNDX:
			f.InputSections = append(f.InputSections, nil)
			break
		default:
			iName := ElfGetName(f.ShStrTab, hdr.Name)
			iContent := f.GetBytesFromIdx(i)
			iSection := NewInputSection(f, iContent, i, &f.ElfSecHdrs[i], iName)
			iSection.SetInputSectionSize(hdr.Size)
			iSection.SetP2Align(hdr.AddrAlign)
			f.InputSections = append(f.InputSections, iSection)
		}
		i += 1
	}
}

func (f *ObjectFile) MarkLiveObjects(ctx *Context, roots []*ObjectFile) []*ObjectFile {
	for i := f.FirstGlobal; i < f.TotalSyms; i++ {
		esym := f.ElfSyms[i]
		sym := f.Symbols[i]
		// necessary, not all global undefined variables
		// are in files?
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

// fill in MergableSections in file
// and fill in fragments in Mergeable sections
// the corresponding input section no longer exist
func (f *ObjectFile) ParseMergeableSections(ctx *Context) {
	f.MergeableSections = make([]*MergeableSection, f.TotalSecs)
	var i uint32
	for _, iSec := range f.InputSections {
		if iSec == nil {
			i += 1
			continue
		}
		if iSec.IsAlive && (iSec.Shdr.Flags&uint64(elf.SHF_MERGE) != 0) {
			f.MergeableSections[i] = f.SplitSection(ctx, iSec)
			iSec.IsAlive = false // this input section no longer used
		}
		i += 1
	}
}

// mergeable sections have two types: strings/constants
// fill in fragOffsets, strs (raw data), and fragments
func (f *ObjectFile) SplitSection(ctx *Context, iSec *InputSection) *MergeableSection {
	m := &MergeableSection{}
	m.OutputSection = ctx.GetMergedSection(iSec) // find using name, type, and flag, if not found create one
	m.P2Align = iSec.P2Align
	m.Fragments = make([]*SectionFragment, 0)

	data := iSec.Content
	shdr := iSec.Shdr

	if (shdr.Flags & uint64(elf.SHF_STRINGS)) != 0 {
		// strings
		var start uint64
		for start < shdr.Size {
			m.FragOffsets = append(m.FragOffsets, start)
			end, found := utils.FindNull(data, start, shdr.Size, int(shdr.EntSize))
			if !found {
				utils.Fatal("Invalid string with no terminate null")
			}
			subStr := string(data[start : end+shdr.EntSize])
			m.Strs = append(m.Strs, subStr)
			frag := m.OutputSection.Insert(subStr, m.P2Align)
			m.Fragments = append(m.Fragments, frag)
			start = end + shdr.EntSize
		}
	} else {
		// constants
		utils.Assert(shdr.Size%shdr.EntSize == 0)
		var start uint64
		for start < shdr.Size {
			m.FragOffsets = append(m.FragOffsets, start)
			subStr := string(data[start : start+shdr.EntSize])
			m.Strs = append(m.Strs, subStr)
			frag := m.OutputSection.Insert(subStr, m.P2Align)
			m.Fragments = append(m.Fragments, frag)
			start += shdr.EntSize
		}
	}

	return m
}

// symbol value is relative to the section
// we fill in the specific fragment and the offset within the fragment
// we can think of fragments as smaller sections and we no longer use the original large sections (before split)
func (f *ObjectFile) ChangeMSecsSymbolsSection() {
	var i uint32
	for i < f.TotalSyms {
		esym := f.ElfSyms[i]
		sym := f.Symbols[i]
		if esym.IsUndef() || esym.IsAbs() || esym.IsCommon() {
			i += 1
			continue
		}
		mSec := f.MergeableSections[esym.GetShndx(f.SymtabShndxSec, i)]
		if mSec == nil {
			i += 1
			continue
		}
		frag, fragOffset := mSec.GetFragment(esym.Val) // return offset within the fragment
		if frag == nil {
			utils.Fatal("Symbol not in fragment")
		}
		sym.SetSectionFragment(frag)
		sym.Value = fragOffset
		i += 1
	}
}
