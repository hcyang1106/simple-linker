package linker

import (
	"debug/elf"
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

type MachineType uint8

const (
	MachineTypeNone MachineType = iota
	MachineTypeRISCV64
)

func (m *MachineType) String() string {
	switch *m {
	case MachineTypeNone:
		return "none"
	case MachineTypeRISCV64:
		return "riscv64"
	}

	utils.Fatal("Invalid machine type")
	return ""
}

func GetMachineTypeFromContent(content []byte) MachineType {
	fileType := GetFileTypeFromContent(content)
	switch fileType {
	case FileTypeObject:
		var machineType uint16
		utils.Read[uint16](content[18:], &machineType)
		switch elf.Machine(machineType) {
		case elf.EM_RISCV:
			class := content[4]
			switch elf.Class(class) {
			case elf.ELFCLASS64:
				return MachineTypeRISCV64
			}
		}
	}

	return MachineTypeNone
}