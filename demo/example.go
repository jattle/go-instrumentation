package demo

import (
	gonativectx "context"
	"go/ast"
	"go/parser"
	"go/token"
	"runtime/trace"
)

// nolint
//
//instrument:exclude
func parseFunc1(filename, functionname string) (fun *ast.FuncDecl, fset *token.FileSet) {
	fset = token.NewFileSet()
	// comment 1
	if file, err := parser.ParseFile(fset, filename, nil, 0); err == nil {
		// comment 2
		for _, d := range file.Decls {
			if f, ok := d.(*ast.FuncDecl); ok && f.Name.Name == functionname {
				// comment 3
				fun = f
				return
			}
			// comment 4
		}
	}
	// replace local vars, function args, names return vars
	panic("function not found")
}

// ProcessFunc instrumentation function example using go trace
func ProcessFunc(spanName string, hasCtx bool, ctx gonativectx.Context, args ...interface{}) {
	fctx := gonativectx.TODO()
	if hasCtx {
		fctx = ctx
	}
	_, t := trace.NewTask(fctx, spanName)
	defer t.End()
}
