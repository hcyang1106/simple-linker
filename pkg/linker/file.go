package linker

import (
	"os"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

type File struct {
	Name string
	Content []byte
	Parent *File
}

func NewFile(filename string) *File {
	content, err := os.ReadFile(filename)
	utils.MustNo(err)
	return &File{
		Name: filename,
		Content: content,
	}
}

func NewFileNoFatal(filename string) *File {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	return &File{
		Name: filename,
		Content: content,
	}
}