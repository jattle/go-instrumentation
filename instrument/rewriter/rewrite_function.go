package rewriter

import (
	"go/ast"
	"go/token"

	"github.com/jattle/go-instrumentation/instrument/parser"
	"github.com/jattle/go-instrumentation/instrument/printer"
)

func rewriteSourceFunc(spanName string, srcMeta parser.FileMeta,
	sourceFunc, patchFunc *ast.FuncDecl) (edits []Edit, err error) {
	if sourceFunc.Body == nil {
		return
	}
	// insert init part of this patch function into begin of source function body
	// patch function:
	// 	ProcessFunc(spanName string, hasCtx bool, ctx context.Context, args ...interface{})
	// auto generated code snippet for it:
	// 	spanNameSuffix := spanName
	// 	if has ctx param:
	// 		   hasCtxSuffix := true
	// 	else hasCtxSuffix = false
	// 	argsSuffix := []interface{}{ctx, args...}
	initStmts := make([]ast.Stmt, 0, 4)
	// always add span stmt
	initStmts = append(initStmts, createSpanStmt(spanName, patchFunc))
	// add hasCtxSuffix := boolean if patchFunc do not ignore this param
	if stmt := createHasCtxDefStmt(sourceFunc, patchFunc); stmt != nil {
		initStmts = append(initStmts, stmt)
	}
	// add ctxSuffix := ctx if patchFunc do not ignore this param
	if stmt := createPatchCtxDefStmt(sourceFunc, patchFunc); stmt != nil {
		initStmts = append(initStmts, stmt)
	}
	// add  argsSuffix := []interface{}{ctx, args...} if patchFunc do not ignore param args
	if stmt := createArgsDefStmt(sourceFunc, patchFunc); stmt != nil {
		initStmts = append(initStmts, stmt)
	}
	blocks := make([]ast.Stmt, 0, len(initStmts)+len(patchFunc.Body.List))
	blocks = append(append(blocks, initStmts...), patchFunc.Body.List...)
	// add ctx = ctxSuffix if source ctx exists and is not ignored by patchFunc, so ctx values can propagate
	if sourceCtxStmt := createSourceCtxAssignStmt(sourceFunc, patchFunc); sourceCtxStmt != nil {
		blocks = append(blocks, sourceCtxStmt)
	}
	var astBytes []byte
	// function block stmts, indented by 1 tab
	// NOTE: comments in patchFunc would be dropped after printing ast node
	astBytes, err = printer.PrintAstNode(blocks, 1)
	if err != nil {
		return
	}
	// token pos is comapacted, get exact bytes offset here
	pos := srcMeta.FSet.Position(sourceFunc.Body.Lbrace).Offset + 1
	edit := Edit{
		OpType:   EditTypeAdd,
		BeginPos: pos,
		EndPos:   pos,
		Content:  astBytes,
	}
	edits = append(edits, edit)
	return
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
					X:   ast.NewIdent("gonativectx"),
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
					Elt: &ast.InterfaceType{
						Methods: &ast.FieldList{},
					},
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
