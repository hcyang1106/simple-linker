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
	Assert(len(content)%size == 0)
	ret := make([]T, 0)
	for len(content) > 0 {
		var ele T
		Read[T](content, &ele)
		ret = append(ret, ele)
		content = content[size:]
	}
	return ret
}

func CountZeros(input uint64) uint8 {
	count := uint8(0)
	for input != 1 {
		count += 1
		input = input >> 1
	}
	Assert(input == 1)
	return count
}

// return start of NULL
func FindNull(data []byte, start uint64, end uint64, size int) (uint64, bool) {
	for start < end {
		curr := data[start : start+uint64(size)]
		if IsNull(curr, size) {
			return start, true
		}
		start += uint64(size)
	}
	return 0, false
}

func IsNull(curr []byte, size int) bool {
	for i := 0; i < size; i++ {
		if curr[i] != 0 {
			return false
		}
	}
	return true
}

func AlignTo(val, align uint64) uint64 {
	if align <= 1 {
		return val
	}

	return (val + align - 1) &^ (align - 1) // do bitwise not, then bitwise and
}
