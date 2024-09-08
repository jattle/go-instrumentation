package rewriter

import (
	"go/ast"
	"go/token"

	"github.com/jattle/go-instrumentation/instrument/parser"
	"github.com/jattle/go-instrumentation/instrument/printer"
)

func getImportDecls(source parser.FileMeta) []*ast.GenDecl {
	decls := make([]*ast.GenDecl, 0)
	for _, decl := range source.ASTFile.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			decls = append(decls, genDecl)
		}
	}
	return decls
}

func createNewImportDecl(source parser.FileMeta, patches []parser.FileMeta) (edit Edit, err error) {
	decl := ast.GenDecl{Tok: token.IMPORT}
	for _, patch := range patches {
		patchImportDecls := getImportDecls(patch)
		traverseDeclSpecs(patchImportDecls, func(spec ast.Spec) {
			decl.Specs = append(decl.Specs, spec)
		})
	}
	// source file has no imports, we could simply place auto-generated imports just below package xxx
	// if we delete all old imports, set offset as the lowest offset
	edit, err = newImportEdit(source, &decl)
	return
}

func newImportEdit(source parser.FileMeta, decl *ast.GenDecl) (edit Edit, err error) {
	newImportOffset := source.FSet.Position(source.ASTFile.Name.Pos()).Offset + len(source.ASTFile.Name.Name) + 1
	edit.OpType = EditTypeAdd
	edit.BeginPos = newImportOffset
	edit.EndPos = newImportOffset
	edit.Content, err = printer.PrintAstNode(decl, 0)
	return
}

type importMeta struct {
	name, path string
}

func insertSpec(importsMap map[importMeta]struct{}, spec *ast.ImportSpec) bool {
	var name string
	if spec.Name != nil {
		name = spec.Name.Name
	}
	meta := importMeta{name: name, path: spec.Path.Value}
	if _, ok := importsMap[meta]; !ok {
		importsMap[meta] = struct{}{}
		return true
	}
	return false
}

func traverseDeclSpecs(decls []*ast.GenDecl, f func(spec ast.Spec)) {
	for _, decl := range decls {
		for _, spec := range decl.Specs {
			f(spec)
		}
	}
}

func mergeImports(source parser.FileMeta, patches []parser.FileMeta) (edits []Edit, err error) {
	// three cases
	// 1. source file has no imports --> create
	// 2. source file has many separate imports, multiline --> add new
	// 3. source file has only one import
	//    one spec --> merge
	//    multi spec --> add before ')'
	sourceImportDecls := getImportDecls(source)
	var edit Edit
	if len(sourceImportDecls) == 0 {
		edit, err = createNewImportDecl(source, patches)
	} else {
		importsMap := make(map[importMeta]struct{})
		traverseDeclSpecs(sourceImportDecls, func(spec ast.Spec) {
			insertSpec(importsMap, spec.(*ast.ImportSpec))
		})
		additionalImportDecl := ast.GenDecl{Tok: token.IMPORT}
		for _, patch := range patches {
			patchImportDecls := getImportDecls(patch)
			traverseDeclSpecs(patchImportDecls, func(spec ast.Spec) {
				// two import specs equal if both name and path equal
				if insertSpec(importsMap, spec.(*ast.ImportSpec)) {
					// only save import not in source file
					additionalImportDecl.Specs = append(additionalImportDecl.Specs, spec)
				}
			})
		}
		// single import, import "xxx" or import ()
		if len(sourceImportDecls) == 1 {
			if len(sourceImportDecls[0].Specs) == 1 {
				// import "xxx"
				// add below package xxx
				edit, err = newImportEdit(source, &additionalImportDecl)
			} else {
				// import ()
				edit.OpType = EditTypeAdd
				pos := source.FSet.Position(sourceImportDecls[0].Rparen).Offset
				edit.BeginPos = pos
				edit.EndPos = pos
				// add before )
				edit.Content, err = printer.PrintAstNodes(additionalImportDecl.Specs, 1)
			}
		} else {
			// multi-imports
			// import "a"
			// import "b"
			// import "c"
			// add below package xxx
			edit, err = newImportEdit(source, &additionalImportDecl)
		}
	}
	if err != nil {
		return
	}
	// imports not need indent
	edits = append(edits, edit)
	return
}
