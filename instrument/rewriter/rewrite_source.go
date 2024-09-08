package rewriter

import (
	"fmt"
	"go/ast"
	"path"
	"sort"

	"github.com/jattle/go-instrumentation/instrument/filter"
	"github.com/jattle/go-instrumentation/instrument/parser"
)

// RewriteSourceFile for every patch file, patch instrumenter func to source file ast,
// for each patch one edition for source code is generated, both for function and imports, finally all editions
// will be applied for this file, source file content will be merged with edited contents.
func RewriteSourceFile(source *parser.FileMeta, patches []parser.FileMeta) error {
	// collect patch funcs
	patchFuncs := make([]*ast.FuncDecl, 0, len(patches))
	for i := range patches {
		if funcDecls, err := RewritePatchASTFunc(patches[i]); err == nil {
			patchFuncs = append(patchFuncs, funcDecls...)
		}
	}
	if len(patchFuncs) == 0 {
		return fmt.Errorf("no valid patch func found")
	}
	sourceFuncs := getFuncDecls(source.ASTFile.Decls)
	// cant find any function declaration, do not need to rewrite
	if len(sourceFuncs) == 0 {
		return nil
	}
	var edits []Edit
	var rewriteNum int
	for _, funcDecl := range sourceFuncs {
		if !filter.DefaultFuncFilter()(funcDecl) {
			continue
		}
		rewriteNum++
		for _, patchFunc := range patchFuncs {
			// spanName = filename - pkg.function
			spanName := genSpanName(source.FileName, source.ASTFile.Name.Name, funcDecl)
			es, err := rewriteSourceFunc(spanName, *source, funcDecl, patchFunc)
			if err != nil {
				return err
			}
			edits = append(edits, es...)
		}
	}
	if rewriteNum > 0 {
		// merge imports
		es, err := mergeImports(*source, patches)
		if err != nil {
			return err
		}
		edits = append(edits, es...)
		// sort apply edits
		sort.Stable(EditSlice(edits))
		rewriter := &FileRewriter{Content: source.Content, Edits: edits}
		if source.Content, err = rewriter.Rewrite(); err != nil {
			return err
		}
	}
	return nil
}

func getFuncDecls(decls []ast.Decl) []*ast.FuncDecl {
	return filter.SelectFuncDecls(decls, func(*ast.FuncDecl) bool { return true })
}

func genSpanName(filename, pkgName string, funcDecl *ast.FuncDecl) string {
	return fmt.Sprintf("%s-%s.%s", path.Base(filename), pkgName, qualifiedFuncName(funcDecl))
}

// qualifiedFuncName extract qualified function name for function
func qualifiedFuncName(funcDecl *ast.FuncDecl) string {
	var (
		prefix string
		suffix = funcDecl.Name.Name
	)
	// has receiver
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		f := funcDecl.Recv.List[0]
		switch t := f.Type.(type) {
		case *ast.Ident:
			// value receiver
			prefix = t.Name
		case *ast.IndexExpr:
			// generic func, func(r *Ring[T]) foo()
			prefix = "*" + t.X.(*ast.Ident).Name
		case *ast.StarExpr:
			// pointer receiver
			switch x := t.X.(type) {
			case *ast.Ident:
				prefix = "*" + x.Name
			case *ast.IndexExpr:
				// generic func, func(r *Ring[T]) foo()
				prefix = "*" + x.X.(*ast.Ident).Name
			}
		default:
			panic(fmt.Sprintf("unknown recv type: %T, field: %+v", t, f))
		}
	}
	if prefix != "" {
		prefix = "(" + prefix + ")."
	}
	return prefix + suffix
}
