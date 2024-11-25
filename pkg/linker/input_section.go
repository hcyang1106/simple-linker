package linker

type InputSection struct {
	ObjFile *ObjectFile
	Content []byte
	Shndx uint32
}

func NewInputSection(obj *ObjectFile, content []byte, shndx uint32) *InputSection {
	return &InputSection {
		ObjFile: obj,
		Content: content,
		Shndx: shndx,
	}
}
