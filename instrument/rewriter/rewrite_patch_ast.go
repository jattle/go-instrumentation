package rewriter

import (
	"fmt"
	"go/ast"

	"github.com/jattle/go-instrumentation/instrument/filter"
	"github.com/jattle/go-instrumentation/instrument/parser"
	"github.com/jattle/go-instrumentation/internal/instrument/astvisitor"
)

// RewritePatchASTFunc rewrite patch file ast, mainly replace local vars, function args, names return vars
func RewritePatchASTFunc(patch parser.FileMeta) (instrumenterFuncs []*ast.FuncDecl, err error) {
	instrumenterFuncs = filter.SelectInstrumentFuncDecls(patch.ASTFile.Decls)
	if len(instrumenterFuncs) == 0 {
		err = fmt.Errorf("instrument func decl not found")
		return
	}
	for _, decl := range instrumenterFuncs {
		varMappings := genFuncVarNameMapping(patch, decl)
		if err = renameFuncVars(decl, varMappings); err != nil {
			return
		}
	}
	return
}

func genFuncVarNameMapping(meta parser.FileMeta, decl *ast.FuncDecl) map[string]string {
	vars, _ := astvisitor.CollectFuncVars(decl)
	varMappings := make(map[string]string)
	suffix := astvisitor.GenVarSuffix(meta.FileName)
	for k := range vars {
		varMappings[k] = k + suffix
	}
	return varMappings
}

func renameFuncVars(funcDecl *ast.FuncDecl, vars map[string]string) error {
	ast.Inspect(funcDecl, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.Ident:
			// blank var should not be renamed
			if v, ok := vars[n.Name]; ok && !isBlankIdent(n.Name) {
				n.Name = v
			}
		}
		return true
	})
	return nil
}

func isBlankIdent(name string) bool {
	const blank = "_"
	return name == blank
}
