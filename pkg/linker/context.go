package linker

type Args struct {
	Output string
}

type Context struct {
	Args Args
}

func NewContext() *Context {
	return &Context {
		Args {
			Output: "a.out",
		},
	}
}
