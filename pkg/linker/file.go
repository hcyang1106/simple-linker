package linker

import (
	"os"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

type File struct {
	Name string
	Content []byte
}

func NewFile(filename string) *File {
	content, err := os.ReadFile(filename)
	utils.MustNo(err)
	return &File{
		Name: filename,
		Content: content,
	}
}