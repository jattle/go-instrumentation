package instrument

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
)

// RewriteSourceFile for every patch file, patch instrumenter func to source file
func RewriteSourceFile(source FileMeta, patches []FileMeta) error {
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
	var rewriteNum int
	for _, funcDecl := range sourceFuncs {
		if !defaultFuncFilter(funcDecl) {
			continue
		}
		rewriteNum++
		for _, patchFunc := range patchFuncs {
			// spanName = filename - pkg.function
			spanName := genSpanName(source.FileName, source.ASTFile.Name.Name, funcDecl)
			rewriteSourceFunc(spanName, funcDecl, patchFunc)
		}
	}
	if rewriteNum > 0 {
		// merge imports
		mergeImports(source, patches)
	}
	return nil
}

func getFuncDecls(decls []ast.Decl) []*ast.FuncDecl {
	return SelectFuncDecls(decls, func(*ast.FuncDecl) bool { return true })
}

// RewritePatchASTFunc rewrite patch file ast, mainly replace local vars, function args, names return vars
func RewritePatchASTFunc(patch FileMeta) (instrumenterFuncs []*ast.FuncDecl, err error) {
	instrumenterFuncs = SelectInstrumentFuncDecls(patch.ASTFile.Decls)
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

// ASTToString convert ast to code
func ASTToString(meta FileMeta) (string, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, meta.FSet, meta.ASTFile); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func genFuncVarNameMapping(meta FileMeta, decl *ast.FuncDecl) map[string]string {
	vars, _ := collectFuncVars(decl)
	varMappings := make(map[string]string)
	suffix := GenVarSuffix(meta.FileName)
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

func getImportDecl(source FileMeta) *ast.GenDecl {
	for _, decl := range source.ASTFile.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			return genDecl
		}
	}
	return nil
}

func mergeImports(source FileMeta, patches []FileMeta) error {
	type importMeta struct {
		name, path string
	}
	importsMap := make(map[importMeta]struct{})
	sourceImportDecl := getImportDecl(source)
	if sourceImportDecl == nil {
		sourceImportDecl = &ast.GenDecl{Tok: token.IMPORT}
		source.ASTFile.Decls = append([]ast.Decl{sourceImportDecl}, source.ASTFile.Decls...)
	}

	putIntoMap := func(spec *ast.ImportSpec) bool {
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
	for _, spec := range sourceImportDecl.Specs {
		putIntoMap(spec.(*ast.ImportSpec))
	}
	for _, patch := range patches {
		patchImportDecl := getImportDecl(patch)
		if patchImportDecl == nil {
			continue
		}
		for _, spec := range patchImportDecl.Specs {
			// two import specs equal if both name and path equal
			importSpec := spec.(*ast.ImportSpec)
			if putIntoMap(importSpec) {
				sourceImportDecl.Specs = append(sourceImportDecl.Specs, importSpec)
			}
		}
	}
	return nil
}

func genSpanName(filename, pkgName string, funcDecl *ast.FuncDecl) string {
	return fmt.Sprintf("%s-%s.%s", BaseFileName(filename), pkgName, qualifiedFuncName(funcDecl))
}

// qualifiedFuncName extract qualified function name for function
func qualifiedFuncName(funcDecl *ast.FuncDecl) string {
	var (
		prefix string = ""
		suffix        = funcDecl.Name.Name
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

func getCtxParamName(decl *ast.FuncDecl) string {
	for _, field := range decl.Type.Params.List {
		sel, ok := field.Type.(*ast.SelectorExpr)
		if ok && sel.X.(*ast.Ident).Name == "context" && sel.Sel.Name == "Context" {
			// func abc(context.Context,string) is valid
			if len(field.Names) > 0 {
				return field.Names[0].Name
			}
		}
	}
	return ""
}

func createSpanStmt(spanName string, patchFunc *ast.FuncDecl) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{
			ast.NewIdent(patchFunc.Type.Params.List[0].Names[0].Name),
		},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.BasicLit{Kind: token.STRING, Value: "\"" + spanName + "\""},
		},
	}
}

func isBlankIdent(name string) bool {
	const blank = "_"
	return name == blank
}

func createHasCtxDefStmt(source *ast.FuncDecl, patchFunc *ast.FuncDecl) *ast.AssignStmt {
	paramNames := patchFunc.Type.Params.List[1].Names
	if len(paramNames) == 0 || isBlankIdent(paramNames[0].Name) {
		return nil
	}
	hasCtxVal := "true"
	ctxParamName := getCtxParamName(source)
	if ctxParamName == "" || isBlankIdent(ctxParamName) {
		hasCtxVal = "false"
	}
	return &ast.AssignStmt{
		Lhs: []ast.Expr{
			ast.NewIdent(paramNames[0].Name),
		},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			ast.NewIdent(hasCtxVal),
		},
	}
}

// createPatchCtxDefStmt create ctx assign stmt for source function if patch func do not ignore ctx param
func createPatchCtxDefStmt(source, patchFunc *ast.FuncDecl) *ast.AssignStmt {
	paramNames := patchFunc.Type.Params.List[2].Names
	if len(paramNames) == 0 || isBlankIdent(paramNames[0].Name) {
		return nil
	}
	// source: has ctx
	// source: no ctx
	sourceCtxName := getCtxParamName(source)
	ctxAssignStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{
			ast.NewIdent(paramNames[0].Name),
		},
		Tok: token.DEFINE,
	}
	if sourceCtxName != "" && !isBlankIdent(sourceCtxName) {
		ctxAssignStmt.Rhs = []ast.Expr{
			ast.NewIdent(sourceCtxName),
		}
	} else {
		ctxAssignStmt.Rhs = []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("context"),
					Sel: ast.NewIdent("Background"),
				},
			},
		}
	}
	return ctxAssignStmt
}

func createArgsDefStmt(source, patchFunc *ast.FuncDecl) *ast.AssignStmt {
	paramNames := patchFunc.Type.Params.List[3].Names
	if len(paramNames) == 0 || isBlankIdent(paramNames[0].Name) {
		return nil
	}

	var elts []ast.Expr
	for _, field := range source.Type.Params.List {
		for _, name := range field.Names {
			if isBlankIdent(name.Name) {
				continue
			}
			elts = append(elts, ast.NewIdent(name.Name))
		}
	}
	return &ast.AssignStmt{
		Lhs: []ast.Expr{
			ast.NewIdent(paramNames[0].Name),
		},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.CompositeLit{
				Type: &ast.ArrayType{
					Elt: ast.NewIdent("any"),
				},
				Elts: elts,
			},
		},
	}
}

// createSourceCtxAssignStmt create source ctx assign stmt if patch func do not ignore ctx param
func createSourceCtxAssignStmt(source, patchFunc *ast.FuncDecl) *ast.AssignStmt {
	paramNames := patchFunc.Type.Params.List[2].Names
	if len(paramNames) == 0 || isBlankIdent(paramNames[0].Name) {
		return nil
	}
	// source: has ctx
	// source: no ctx
	sourceCtxName := getCtxParamName(source)
	if sourceCtxName == "" || isBlankIdent(sourceCtxName) {
		return nil
	}
	ctxAssignStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{
			ast.NewIdent(sourceCtxName),
		},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{
			ast.NewIdent(paramNames[0].Name),
		},
	}
	return ctxAssignStmt
}

func rewriteSourceFunc(spanName string, source, patchFunc *ast.FuncDecl) error {
	// insert init part for this patch function
	// 	spanNameSuffix := spanName
	// 	if has ctx param:
	// 		   hasCtxSuffix := true
	// 	else hasCtxSuffix = false
	// 	argsSuffix := []interface{}{ctx, args...}
	// 	ProcessFunc(spanName string, hasCtx bool, ctx context.Context, args ...any)
	// generate init part
	initStmts := make([]ast.Stmt, 0, 4)
	// always add span stmt
	initStmts = append(initStmts, createSpanStmt(spanName, patchFunc))
	// add hasCtxSuffix := boolean if patchFunc do not ignore this param
	if stmt := createHasCtxDefStmt(source, patchFunc); stmt != nil {
		initStmts = append(initStmts, stmt)
	}
	// add ctxSuffix := ctx if patchFunc do not ignore this param
	if stmt := createPatchCtxDefStmt(source, patchFunc); stmt != nil {
		initStmts = append(initStmts, stmt)
	}
	// add  argsSuffix := []interface{}{ctx, args...} if patchFunc do not ignore param args
	if stmt := createArgsDefStmt(source, patchFunc); stmt != nil {
		initStmts = append(initStmts, stmt)
	}
	blocks := make([]ast.Stmt, 0, len(initStmts)+len(patchFunc.Body.List))
	blocks = append(append(blocks, initStmts...), patchFunc.Body.List...)
	// add ctx = ctxSuffix if source ctx exists and is not ignored by patchFunc, so ctx values can propagate
	if sourceCtxStmt := createSourceCtxAssignStmt(source, patchFunc); sourceCtxStmt != nil {
		blocks = append(blocks, sourceCtxStmt)
	}
	source.Body.List = append(append([]ast.Stmt(nil), blocks...), source.Body.List...)

	return nil
}