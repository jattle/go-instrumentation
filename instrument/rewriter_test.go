package instrument

import (
	"go/ast"
	"testing"

	"gotest.tools/assert"
)

func getFuncByName(name string) FuncFilter {
	return func(decl *ast.FuncDecl) bool {
		return decl.Name.Name == name
	}
}

func TestRewritePatchASTFunc(t *testing.T) {
	meta, err := ParseFile("../demo/example.go")
	if err != nil {
		t.Error(err)
		return
	}

	ifuncs, err := RewritePatchASTFunc(meta)
	assert.NilError(t, err)
	// func ProcessFunc(spanName string, hasCtx bool, ctx context.Context, args ...interface{}) {
	// params of processFunc was modified
	assert.Equal(t, len(ifuncs), 1)
	param1 := ifuncs[0].Type.Params.List[0].Names[0].Name
	param2 := ifuncs[0].Type.Params.List[1].Names[0].Name
	param3 := ifuncs[0].Type.Params.List[2].Names[0].Name
	param4 := ifuncs[0].Type.Params.List[3].Names[0].Name
	assert.Assert(t, param1 != "spanName")
	assert.Assert(t, param2 != "hasCtx")
	assert.Assert(t, param3 != "ctx")
	assert.Assert(t, param4 != "args")

	// params of parseFunc1 was not modified
	funcs := SelectFuncDecls(meta.ASTFile.Decls, getFuncByName("parseFunc1"))
	assert.Equal(t, len(funcs), 1)
	param1 = funcs[0].Type.Params.List[0].Names[0].Name
	param2 = funcs[0].Type.Params.List[0].Names[1].Name
	assert.Equal(t, param1, "filename")
	assert.Equal(t, param2, "functionname")
}
