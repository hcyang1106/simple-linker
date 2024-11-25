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

// o => -o
// plugin => -plugin, --plugin
func AddDashes(option string) []string {
	res := []string{}

	if len(option) == 1 {
		res = append(res, "-"+option)
	} else {
		res = append(res, "-"+option, "--"+option)
	}

	return res
}

func ReadSlice[T any](content []byte, size int) []T {
	Assert(len(content) % size == 0)
	ret := make([]T, 0)
	for len(content) > 0 {
		var ele T
		Read[T](content, &ele)
		ret = append(ret, ele)
		content = content[size:]
	}
	return ret
}