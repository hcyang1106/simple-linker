package linker

import (
	"debug/elf"
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
	Args           Args
	SymbolMap      map[string]*Symbol
	MergedSections []*MergedSection
	InternalObj    *ObjectFile
	InternalEsyms  []Sym
	Buf []byte
	Ehdr *OutputEhdr
}

func NewContext() *Context {
	return &Context{
		Args: Args{
			Output:  "a.out",
			Machine: MachineTypeNone,
		},
		SymbolMap: make(map[string]*Symbol),
	}
}

func (c *Context) AddSymbol(name string, symbol *Symbol) {
	c.SymbolMap[name] = symbol
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
			// this part is necessary
			// otherwise the code below this part
			// will get the wrong argument (with =)
			// -melf64.... versus -plugin-opt=XXX
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

// note that obj files in archive files are not alive
// and those are in archive files are alive
func (c *Context) FillInObjFiles(remaining []string) {
	for _, name := range remaining {
		// lib file
		if strings.HasPrefix(name, "-l") {
			libName := "lib" + name[2:] + ".a"
			files := c.readArchiveMembers(libName)
			for _, file := range files {
				utils.Assert(GetFileTypeFromContent(file.Content) == FileTypeObject)
				CheckFileCompatibility(c, file)
				NewObjectFile(file, false, c)
			}
			continue
		}

		file := NewFile(name)
		CheckFileCompatibility(c, file)
		NewObjectFile(file, true, c)
	}
}

// -L specifies the library path, and -l specifies the filename
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
			Parent:  file,
		})

		pos += arHdr.GetSize()
	}
	return ret
}

func (c *Context) GetSymbol(name string) *Symbol {
	if sym, ok := c.SymbolMap[name]; ok {
		return sym
	}
	c.SymbolMap[name] = NewSymbol(nil, "") // for file with definition ot overwrite
	return c.SymbolMap[name]
}

// bitwise not => ^
// bitwise negation => ~
// only mergeable sections with same name, flag, and type could be merged
func (c *Context) GetMergedSection(iSec *InputSection) *MergedSection {
	outName := iSec.GetOutputSectionName()
	flags := iSec.Shdr.Flags & ^uint64(elf.SHF_GROUP) & ^uint64(elf.SHF_MERGE) &
		^uint64(elf.SHF_STRINGS) & ^uint64(elf.SHF_COMPRESSED)
	typ := iSec.Shdr.Type

	var ret *MergedSection
	for _, mSec := range c.MergedSections {
		if mSec.Name != outName || mSec.Shdr.Flags != flags ||
			mSec.Shdr.Type != typ {
			continue
		}
		ret = mSec
	}

	if ret != nil {
		return ret
	}

	newMSec := NewMergedSection(outName, flags, typ)
	c.MergedSections = append(c.MergedSections, newMSec)
	return newMSec
}

// not initialized
func (ctx *Context) CreateInternalFile() {
	obj := &ObjectFile{}
	ctx.InternalObj = obj
	ctx.Args.ObjFiles = append(ctx.Args.ObjFiles, obj)
	// first symbol is empty
	ctx.InternalEsyms = make([]Sym, 1)
	obj.Symbols = append(ctx.InternalObj.Symbols, NewSymbol(obj, ""))
	obj.IsAlive = true
	obj.FirstGlobal = 1
	obj.ElfSyms = ctx.InternalEsyms
}
