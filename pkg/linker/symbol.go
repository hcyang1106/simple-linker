package linker

const (
	IsInGot uint32 = 1 << 0
)

type Symbol struct {
	File            *ObjectFile
	InputSection    *InputSection
	SectionFragment *SectionFragment
	Name            string
	Value           uint64
	SymIdx          uint32
	GotEntryIdx     uint32
	Flags           uint32
}

func NewSymbol(file *ObjectFile, name string) *Symbol {
	return &Symbol{
		File: file,
		Name: name,
	}
}

// either use fragment or input section
func (s *Symbol) SetInputSection(section *InputSection) {
	s.InputSection = section
	s.SectionFragment = nil
}

// either use fragment or input section
func (s *Symbol) SetSectionFragment(frag *SectionFragment) {
	s.SectionFragment = frag
	s.InputSection = nil
}

func (s *Symbol) SetValue(value uint64) {
	s.Value = value
}

func (s *Symbol) SetSymIdx(idx uint32) {
	s.SymIdx = idx
}

func (s *Symbol) GetAddr() uint64 {
	if s.SectionFragment != nil {
		return s.SectionFragment.GetAddr() + s.Value
	}
	if s.InputSection != nil {
		return s.InputSection.GetAddr() + s.Value
	}
	return s.Value
}

func (s *Symbol) GetGotEntryAddr(ctx *Context) uint64 {
	return ctx.OutputGotSectionWriter.Shdr.Addr + uint64(s.GotEntryIdx*8)
}
