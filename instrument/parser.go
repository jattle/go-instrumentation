package instrument

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"sync/atomic"
	"time"
)

// FileMeta file meta
type FileMeta struct {
	FileName string
	FSet     *token.FileSet
	ASTFile  *ast.File
}

var (
	varCounter = atomic.Int32{}
)

// ParseFile parse go source file
func ParseFile(filename string) (meta FileMeta, err error) {
	meta.FileName = filename
	meta.FSet = token.NewFileSet()
	if meta.ASTFile, err = parser.ParseFile(meta.FSet, filename, nil, parser.ParseComments); err != nil {
		err = fmt.Errorf("parse file %s failed: %w", filename, err)
		return
	}
	return
}

// BaseName file base name not with filetype suffix, eg a/b/base.go => base
func BaseName(filename string) string {
	var baseName = BaseFileName(filename)
	if i := strings.LastIndex(baseName, "."); i != -1 {
		baseName = baseName[:i]
	}
	return baseName
}

// BaseFileName base file name, eg a/b/base.go => base.go
func BaseFileName(filename string) string {
	var baseName string = filename
	if i := strings.LastIndex(filename, "/"); i != -1 {
		baseName = filename[i+1:]
	}
	return baseName
}

// GenVarSuffix gen var suffix by patch name
func GenVarSuffix(patch string) string {
	prefix := fmt.Sprintf("%s%d%d", BaseName(patch), time.Now().Unix(), varCounter.Add(1))
	return ToValidVarName(prefix)
}

// ToValidVarName to valid var name
func ToValidVarName(v string) string {
	// remove _
	return strings.ReplaceAll(v, "_", "")
}

type varVistor struct {
	vars []string
}

// Visit collect declared vars
func (f *varVistor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.Ident:
		if n.Obj != nil && n.Obj.Kind == ast.Var {
			f.vars = append(f.vars, n.Name)
		}
	}
	return f
}

func addVarNames(names []string, vars map[string]struct{}) {
	for _, n := range names {
		vars[n] = struct{}{}
	}
}

func collectFuncVars(funcDecl *ast.FuncDecl) (map[string]struct{}, error) {
	m := make(map[string]struct{})
	ast.Inspect(funcDecl, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.GenDecl, *ast.AssignStmt:
			collectNodeVars(n, m)
		case *ast.CallExpr:
			return false
		case *ast.FuncType:
			if n.Params != nil {
				collectNodeVars(n.Params, m)
			}
		}
		//	if n.Results != nil {
		//		collectNodeVars(n.Results, m)
		//	}
		//}
		return true
	})
	return m, nil
}

func collectNodeVars(n ast.Node, m map[string]struct{}) {
	v := varVistor{}
	ast.Walk(&v, n)
	addVarNames(v.vars, m)
}
