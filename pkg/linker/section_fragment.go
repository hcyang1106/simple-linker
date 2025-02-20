package linker

import "math"

type SectionFragment struct {
	OutputSection *MergedSection
	Offset uint32
	P2Align uint8
	IsAlive bool
}

func NewSectionFragment() *SectionFragment {
	return &SectionFragment {
		Offset: math.MaxUint32,
	}
}

func (s *SectionFragment) SetIsAlive(isAlive bool) {
	s.IsAlive = isAlive
}

func (s *SectionFragment) SetOffset(offset uint32) {
	s.Offset = offset
}

func (s *SectionFragment) SetP2Align(align uint8) {
	s.P2Align = align
}

func (s *SectionFragment) SetOutputSection(output *MergedSection) {
	s.OutputSection = output
}

func (s *SectionFragment) GetAddr() uint64 {
	return s.OutputSection.Shdr.Addr + uint64(s.Offset)
}
