package parser

import (
	"fmt"
	"go/ast"
	"path"
	"strings"
	"sync/atomic"
	"time"
)

var (
	varCounter = atomic.Int32{}
)

// BaseName file base name without filetype suffix, eg a/b/base.go => base
func BaseName(filename string) string {
	var baseName = path.Base(filename)
	if i := strings.LastIndex(baseName, "."); i != -1 {
		baseName = baseName[:i]
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

// CollectFuncVars collect function variable names
func CollectFuncVars(funcDecl *ast.FuncDecl) (map[string]struct{}, error) {
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
		return true
	})
	return m, nil
}

func collectNodeVars(n ast.Node, m map[string]struct{}) {
	v := varVistor{}
	ast.Walk(&v, n)
	addVarNames(v.vars, m)
}
