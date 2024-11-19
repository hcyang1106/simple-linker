package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"runtime/debug"
)

func Fatal(v any) {
	fmt.Printf("fatal: %v\n", v)
	debug.PrintStack()
	os.Exit(1)
}

func MustNo(err error) {
	if err != nil {
		Fatal(err)
	}
}

func Read[T any](content []byte, val *T) {
	reader := bytes.NewReader(content)
	err := binary.Read(reader, binary.LittleEndian, val) //RISC-V uses little endian
	MustNo(err)
}

func Assert(res bool) {
	if !res {
		Fatal(res)
	}
}