package linker

import "sort"

type MergeableSection struct {
	OutputSection *MergedSection
	P2Align uint8
	Strs []string
	FragOffsets []uint64
	Fragments []*SectionFragment
}

func (m *MergeableSection) GetFragment(offset uint64) (*SectionFragment, uint64) {
	pos := sort.Search(len(m.FragOffsets), func(i int) bool {
		return offset < m.FragOffsets[i]
	})
	// not found
	if pos == 0 {
		return nil, 0
	}

	idx := pos - 1
	return m.Fragments[idx], offset - m.FragOffsets[idx]
}
