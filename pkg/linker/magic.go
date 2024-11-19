package linker

import (
	"bytes"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

func MustHaveMagic(content []byte) {
	// check magic number
	if !bytes.HasPrefix(content, []byte("\177ELF")) {
		utils.Fatal("Invalid magic number")
	}
}
