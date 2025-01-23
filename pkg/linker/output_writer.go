package linker

type iOutputWriter interface {
	GetShdr() *Shdr
	CopyBuf(ctx *Context)
}

type OutputWriter struct {
	Name string
	Shdr Shdr
}

func NewOutputWriter() *OutputWriter {
	return &OutputWriter {
		Shdr: Shdr {
			AddrAlign: 1,
		},
	}
}

func (o *OutputWriter) GetShdr() *Shdr {
	return &o.Shdr
}

func (o *OutputWriter) CopyBuf(ctx *Context) {

}