package linker

type MergedSection struct {
	Chunk
	Map map[string]*SectionFragment
}

func NewMergedSection (name string, flags uint64, typ uint32) *MergedSection {
	m := &MergedSection {
		Chunk: *NewChunk(),
		Map: make(map[string]*SectionFragment),
	}
	m.Name = name
	m.Shdr.Flags = flags
	m.Shdr.Type = typ
	return m
}

func (m *MergedSection) Insert(key string, p2align uint8) *SectionFragment {
	if frag, ok := m.Map[key]; ok {
		if frag.P2Align < p2align {
			frag.P2Align = p2align
		}
		return frag
	}
	m.Map[key] = NewSectionFragment()
	return m.Map[key]
}