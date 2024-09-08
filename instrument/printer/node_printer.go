package printer

import (
	"bytes"
	"go/printer"
	"go/token"

	"github.com/jattle/go-instrumentation/instrument/parser"
)

// PrintAstNode convert node to code
// node: The node type must be *ast.File, *CommentedNode, []ast.Decl, []ast.Stmt,
// or assignment-compatible to ast.Expr, ast.Decl, ast.Spec, or ast.Stmt.
// indent: code indented by {indent} tab
func PrintAstNode(node any, indent int) ([]byte, error) {
	const (
		tabWidth                = 8
		printerNormalizeNumbers = 1 << 30
		printerMode             = printer.UseSpaces | printer.TabIndent | printerNormalizeNumbers
		// printerNormalizeNumbers means to canonicalize number literal prefixes
	)
	var buf bytes.Buffer
	buf.WriteByte('\n')
	fset := token.NewFileSet()
	var config = printer.Config{Mode: printerMode, Tabwidth: tabWidth, Indent: indent}
	if err := config.Fprint(&buf, fset, node); err != nil {
		return buf.Bytes(), err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// PrintAstNodes print node array, T is node type suitable for PrintAsNode
func PrintAstNodes[T any](nodes []T, indent int) ([]byte, error) {
	buf := bytes.Buffer{}
	for _, n := range nodes {
		b, err := PrintAstNode(n, indent)
		if err != nil {
			return nil, err
		}
		// ignore leading char '\n'
		buf.Write(b[1:])
	}
	return buf.Bytes(), nil
}

// ASTToString convert ast to code
func ASTToString(meta parser.FileMeta) (string, error) {
	buf, err := PrintAstNode(meta.ASTFile, 0)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
