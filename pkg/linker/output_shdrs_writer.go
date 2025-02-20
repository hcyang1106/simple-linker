package linker

import (
	"github.com/hcyang1106/simple-linker/pkg/utils"
)

// should setup size, offset before using
type OutputShdrsWriter struct {
	OutputWriter
}

func NewOutputShdrsWriter() *OutputShdrsWriter {
	return &OutputShdrsWriter{
		OutputWriter{
			Name: "shdr",
			Shdr: Shdr{
				Name: 1000,
				AddrAlign: 8,
			},
		},
	}
}

// should setup ctx.OutputWriters before calling
func (o *OutputShdrsWriter) UpdateSize(ctx *Context) {
	n := uint64(0)
	for _, o := range ctx.OutputWriters {
		if o.GetShndx() > 0 {
			n = uint64(o.GetShndx())
		}
	}
	o.Shdr.Size = (n + 1) * uint64(ShdrSize)
}

func (o *OutputShdrsWriter) CopyBuf(ctx *Context) {
	base := ctx.Buf[o.Shdr.Offset:]
	utils.Write[Shdr](base, Shdr{})

	for _, o := range ctx.OutputWriters {
		if o.GetShndx() > 0 {
			utils.Write[Shdr](base[o.GetShndx()*int64(ShdrSize):],
				*o.GetShdr())
		}
	}
}
