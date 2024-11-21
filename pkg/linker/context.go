package linker

import (
	"fmt"
	"github.com/hcyang1106/simple-linker/pkg/utils"
	"os"
	"strings"
)

type Args struct {
	Output       string
	Machine      MachineType
	LibraryPaths []string
	ObjFiles     []*ObjectFile
}

type Context struct {
	Args Args
}

func NewContext() *Context {
	return &Context{
		Args{
			Output:  "a.out",
			Machine: MachineTypeNone,
		},
	}
}

func (c *Context) ParseArgs(ctx *Context, version string) []string {
	args := os.Args[1:] // ignore ./ld

	// usage: readFlag("help")
	// or readFlag("o")
	readFlag := func(option string) bool {
		for _, opt := range utils.AddDashes(option) {
			if args[0] == opt {
				args = args[1:]
				return true
			}
		}
		return false
	}

	arg := ""
	readOpt := func(option string) bool {
		for _, opt := range utils.AddDashes(option) {
			if args[0] == opt {
				args = args[1:]
				if len(args) == 0 {
					utils.Fatal("No option specified")
				}
				arg = args[0] // get option argument
				args = args[1:]
				return true
			}

			// -plugin-opt=
			if len(option) > 1 {
				opt += "="
			}
			if strings.HasPrefix(args[0], opt) {
				arg = args[0][len(opt):]
				args = args[1:]
				return true
			}
		}
		return false
	}

	remaining := make([]string, 0)
	for len(args) > 0 {
		if readFlag("help") {
			fmt.Printf("usage: %s [options] files...\n", os.Args[0])
			os.Exit(0)
		} else if readOpt("o") || readOpt("output") {
			ctx.Args.Output = arg
		} else if readFlag("v") || readFlag("version") {
			fmt.Printf("simple-linker %s\n", version)
		} else if readOpt("m") {
			if arg == "elf64lriscv" {
				ctx.Args.Machine = MachineTypeRISCV64
			} else {
				utils.Fatal("Unknown -m argument")
			}
		} else if readOpt("L") {
			ctx.Args.LibraryPaths = append(ctx.Args.LibraryPaths, arg)
		} else if readOpt("sysroot") ||
			readOpt("plugin") ||
			readOpt("plugin-opt") ||
			readOpt("hash-style") ||
			readOpt("build-id") ||
			readFlag("static") ||
			readFlag("as-needed") ||
			readFlag("start-group") ||
			readFlag("end-group") ||
			readFlag("s") ||
			readFlag("no-relax") {
			// Ignored
		} else {
			remaining = append(remaining, args[0])
			args = args[1:]
		}

	}

	return remaining
}

func (c *Context) FillInObjFiles(remaining []string) {
	for _, name := range remaining {
		// lib file
		if strings.HasPrefix(name, "-l") {
			libName := "lib" + name[2:] + ".a"
			files := c.readArchiveMembers(libName)
			for _, file := range files {
				utils.Assert(GetFileTypeFromContent(file.Content) == FileTypeObject)
				CheckFileCompatibility(c, file)
				obj := NewObjectFile(file, false)
				c.Args.ObjFiles = append(c.Args.ObjFiles, obj)
			}
			continue
		}

		file := NewFile(name)
		CheckFileCompatibility(c, file)
		obj := NewObjectFile(file, true)
		c.Args.ObjFiles = append(c.Args.ObjFiles, obj)
	}
}

func (c *Context) readArchiveMembers(filename string) []*File {
	var file *File
	for _, path := range c.Args.LibraryPaths {
		file = NewFileNoFatal(path + "/" + filename)
		if file != nil {
			break
		}
	}
	utils.Assert(file != nil)

	ret := make([]*File, 0)
	// first hdr, section, second hdr section....
	// [!<arch>\n][ArHdr][]\n[ArHdr][][ArHdr][][ArHdr][]\n
	// section part is two bytes aligned, if not a \n is added
	content := file.Content
	utils.Assert(GetFileTypeFromContent(content) == FileTypeArchive)
	pos := 8
	var strTab []byte
	for pos < len(content)-1 {
		if pos%2 == 1 {
			pos++
		}
		var arHdr ArHdr
		utils.Read[ArHdr](content[pos:], &arHdr)
		pos += AhdrSize

		if arHdr.IsSymtab() {
			pos += arHdr.GetSize()
			continue
		}
		if arHdr.IsStrTab() {
			strTab = content[pos : pos+arHdr.GetSize()]
			pos += arHdr.GetSize()
			continue
		}

		ret = append(ret, &File{
			Name:    arHdr.ReadName(strTab),
			Content: content[pos : pos+arHdr.GetSize()],
			Parent: file,
		})

		pos += arHdr.GetSize()
	}
	return ret
}
