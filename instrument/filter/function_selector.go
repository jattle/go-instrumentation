package filter

import (
	"go/ast"
	"regexp"
	"strings"
)

// FuncFilter source function filter
type FuncFilter func(*ast.FuncDecl) bool

var (
	FuncNameExcludeExpr *regexp.Regexp
	filters             = []FuncFilter{negateFunc(excludeCommentFilter), negateFunc(funcNameExcludeFilter)}
	defaultFuncFilter   = filterBundle(filters).matchSourceFunc
	// DefaultFuncFilter get unified function filter to select func
	DefaultFuncFilter = func() FuncFilter {
		return defaultFuncFilter
	}
)

type filterBundle []FuncFilter

func (f filterBundle) matchSourceFunc(decl *ast.FuncDecl) bool {
	for _, filter := range f {
		if !filter(decl) {
			return false
		}
	}
	return true
}

func negateFunc(f FuncFilter) FuncFilter {
	return func(decl *ast.FuncDecl) bool {
		return !f(decl)
	}
}

func excludeCommentFilter(decl *ast.FuncDecl) bool {
	const excludeComment = "//instrument:exclude"
	if decl.Doc == nil {
		return false
	}
	for _, comment := range decl.Doc.List {
		if strings.HasPrefix(comment.Text, excludeComment) {
			return true
		}
	}
	return false
}

func funcNameExcludeFilter(decl *ast.FuncDecl) bool {
	if FuncNameExcludeExpr == nil {
		return false
	}
	return matchName(FuncNameExcludeExpr, decl.Name.Name)
}

func matchName(pat *regexp.Regexp, name string) bool {
	return pat.MatchString(name)
}

// SelectInstrumentFuncDecls select instrument func decls
func SelectInstrumentFuncDecls(decls []ast.Decl) []*ast.FuncDecl {
	return SelectFuncDecls(decls, matchInstrumentSignature)
}

// SelectFuncDecls select func decls which match filter
func SelectFuncDecls(decls []ast.Decl, f FuncFilter) []*ast.FuncDecl {
	rets := make([]*ast.FuncDecl, 0, len(decls))
	for _, decl := range decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && f(funcDecl) {
			rets = append(rets, funcDecl)
		}
	}
	return rets
}

func matchInstrumentSignature(decl *ast.FuncDecl) bool {
	// instrument function signature
	// ProcessFunc(spanName string, hasCtx bool, ctx context.Context, args ...interface{})
	params := decl.Type.Params.List
	if len(decl.Type.Params.List) != 4 {
		return false
	}
	if len(params[0].Names) != 1 || params[0].Type.(*ast.Ident).Name != "string" {
		return false
	}
	if len(params[1].Names) != 1 || params[1].Type.(*ast.Ident).Name != "bool" {
		return false
	}
	if len(params[2].Names) != 1 {
		return false
	}
	if sel, ok := params[2].Type.(*ast.SelectorExpr); !ok ||
		sel.X.(*ast.Ident).Name != "gonativectx" || sel.Sel.Name != "Context" {
		return false
	}
	// args ...interface{}
	if len(params[3].Names) != 1 {
		return false
	}
	t, ok := params[3].Type.(*ast.Ellipsis)
	if !ok {
		return false
	}
	it, ok := t.Elt.(*ast.InterfaceType)
	if !ok {
		return false
	}
	if len(it.Methods.List) != 0 {
		return false
	}
	return true
}
