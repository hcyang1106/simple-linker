package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/bits"
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
	err := binary.Read(reader, binary.LittleEndian, val) // RISC-V uses little endian
	MustNo(err)
}

func ReadWithReturn[T any](data []byte) (val T) {
	reader := bytes.NewReader(data)
	err := binary.Read(reader, binary.LittleEndian, &val)
	MustNo(err)
	return
}

func Write[T any](buf []byte, val T) {
	b := &bytes.Buffer{}
	err := binary.Write(b, binary.LittleEndian, val) //RISC-V uses little endian
	MustNo(err)
	copy(buf, b.Bytes())
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
	if align == 0 {
		return val
	}

	return (val + align - 1) &^ (align - 1) // do bitwise not, then bitwise and
}

// ToP2Align converts using counting the zeros at the end
func ToP2Align(align uint64) uint8 {
	if align == 0 {
		return 0
	}
	return uint8(bits.TrailingZeros64(align))
}

func hasSingleBit(n uint64) bool {
	return n&(n-1) == 0
}

func BitCeil(val uint64) uint64 {
	if hasSingleBit(val) {
		return val
	}
	return 1 << (64 - bits.LeadingZeros64(val))
}

type Uint interface {
	uint8 | uint16 | uint32 | uint64
}

func Bit[T Uint](val T, pos int) T {
	return (val >> pos) & 1
}

func Bits[T Uint](val T, hi T, lo T) T {
	return (val >> lo) & ((1 << (hi - lo + 1)) - 1)
}

func SignExtend(val uint64, size int) uint64 {
	return uint64(int64(val<<(63-size)) >> (63 - size))
}

func AllZeros(bs []byte) bool {
	b := byte(0)
	for _, s := range bs {
		b |= s
	}

	return b == 0
}
