package astvisitor

import (
	"go/ast"
	"testing"

	"github.com/jattle/go-instrumentation/instrument/parser"
	"gotest.tools/assert"
)

func TestCollectFuncVars(t *testing.T) {
	content := `
package main

import (
	"context"
	"fmt"
	"runtime/trace"
)

func ProcessFunc(spanName string, hasCtx bool, ctx context.Context, args ...interface{}) {
	fctx := context.Background()
	if hasCtx {
		fctx = ctx
	}
	fctx, t := trace.NewTask(fctx, spanName)
	defer t.End()
	fmt.Println("process func template", fctx)
}
	`
	f, err := parser.ParseContent("example.go", []byte(content))
	assert.Equal(t, err, nil)
	varMap := make(map[string]struct{})
	for _, v := range f.ASTFile.Decls {
		if fun, ok := v.(*ast.FuncDecl); ok {
			m, err := CollectFuncVars(fun)
			assert.Equal(t, err, nil)
			for k, v := range m {
				varMap[k] = v
			}
		}
	}
	names := [...]string{
		"spanName", "hasCtx", "ctx", "args", "fctx", "t",
	}
	assert.Equal(t, len(names), len(varMap))
	for _, n := range names {
		_, ok := varMap[n]
		assert.Equal(t, ok, true)
	}
}
