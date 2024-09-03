package instrument

import (
	"bytes"
	"go/printer"
	"go/token"
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

// ASTToString convert ast to code
func ASTToString(meta FileMeta) (string, error) {
	buf, err := PrintAstNode(meta.ASTFile, 0)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
