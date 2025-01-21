package linker

func MarkLiveObjects(ctx *Context) {
	roots := make([]*ObjectFile, 0)
	for _, file := range ctx.Args.ObjFiles {
		if file.IsAlive {
			roots = append(roots, file)
		}
	}
	for len(roots) > 0 {
		roots = roots[0].MarkLiveObjects(ctx, roots)
		roots = roots[1:]
	}
}

func ClearSymbolsAndFiles(ctx *Context) {
	ClearUnusedGlobalSymbols(ctx)
	ClearUnusedFiles(ctx)
}

func ClearUnusedGlobalSymbols(ctx *Context) {
	for _, file := range ctx.Args.ObjFiles {
		if !file.IsAlive {
			file.ClearUnusedGlobalSymbols(ctx)
		}
	}
}

func ClearUnusedFiles(ctx *Context) {
	var i int = 0
	for _, file := range ctx.Args.ObjFiles {
		if file.IsAlive {
			ctx.Args.ObjFiles[i] = file
			i += 1
		}
	}
	ctx.Args.ObjFiles = ctx.Args.ObjFiles[:i]
}

// change symbol corresponding sections to "segments"
func ChangeMSecsSymbolsSection(ctx *Context) {
	for _, file := range ctx.Args.ObjFiles {
		file.ChangeMSecsSymbolsSection()
	}
}

//func CreateSyntheticSections()