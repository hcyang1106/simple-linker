package linker

import (
	"bytes"
	"debug/elf"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

type FileType uint8

const (
	FileTypeUnknown FileType = iota
	FileTypeEmpty
	FileTypeObject
	FileTypeArchive
)

func GetFileTypeFromContent(content []byte) FileType {
	if len(content) == 0 {
		return FileTypeEmpty
	}
	// ensure it is an elf
	// elf is divided into many types
	if CheckMagic(content) {
		var elfType uint16
		utils.Read[uint16](content[16:], &elfType)
		switch elf.Type(elfType) {
		case elf.ET_REL:
			return FileTypeObject
		}
	}

	if bytes.HasPrefix(content, []byte("!<arch>\n")) {
		return FileTypeArchive
	}

	return FileTypeUnknown
}

func CheckFileCompatibility(ctx *Context, file *File) {
	t := GetMachineTypeFromContent(file.Content)
	if ctx.Args.Machine != t {
		utils.Fatal("Object file is not compatible to machine type")
	}
}