package linker

type Symbol struct {
	File *ObjectFile
	InputSection *InputSection
	Name string
	Value uint64
	SymIdx uint32
}

func NewSymbol(file *ObjectFile, name string) *Symbol {
	return &Symbol {
		File: file,
		Name: name,
	}
}

func (s *Symbol) SetInputSection (section *InputSection){
	s.InputSection = section
}

func (s *Symbol) SetValue (value uint64){
	s.Value = value
}

func (s *Symbol) SetSymIdx (idx uint32){
	s.SymIdx = idx
}