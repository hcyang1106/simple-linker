package linker

import (
	"sort"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

type MergedSection struct {
	OutputWriter
	Map map[string]*SectionFragment
}

func NewMergedSection(name string, flags uint64, typ uint32) *MergedSection {
	m := &MergedSection{
		OutputWriter: *NewOutputWriter(),
		Map:          make(map[string]*SectionFragment),
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
	frag := NewSectionFragment()
	frag.SetOutputSection(m)
	frag.P2Align = p2align
	m.Map[key] = frag
	return m.Map[key]
}

// m.Map stores all the substrings->fragments
func (m *MergedSection) AssignFragmentsOffsets() {
	type f struct {
		Key string
		Val *SectionFragment
	}
	var fragments []f
	for key, val := range m.Map {
		fragments = append(fragments, f{
			Key: key,
			Val: val,
		})
	}

	sortFunc := func (i, j int) bool {
		x := fragments[i]
		y := fragments[j]
		if x.Val.P2Align != y.Val.P2Align {
			return x.Val.P2Align < y.Val.P2Align
		}
		if len(x.Key) != len(y.Key) {
			return len(x.Key) < len(y.Key)
		}
		// alphabetical
		return x.Key < y.Key
	}
	sort.SliceStable(fragments, sortFunc)

	offset := uint64(0)
	p2align := uint64(0)
	for _, frag := range fragments {
		offset = utils.AlignTo(offset, 1<<frag.Val.P2Align)
		frag.Val.Offset = uint32(offset)
		offset += uint64(len(frag.Key))
		if p2align < uint64(frag.Val.P2Align) {
			p2align = uint64(frag.Val.P2Align)
		}
	}

	// not sure why we also make alignment here
	m.Shdr.Size = utils.AlignTo(offset, 1<<p2align)
	m.Shdr.AddrAlign = 1 << p2align
}

func (m *MergedSection) CopyBuf(ctx *Context) {
	start := ctx.Buf[m.Shdr.Offset:]
	for key, frag := range m.Map {
		// no need to align because it is already aligned
		// map is unordered but offsets are already assigned
		copy(start[frag.Offset:], key)
	}
}
