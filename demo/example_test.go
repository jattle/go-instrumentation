package demo

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
	"testing"

	iparser "github.com/jattle/go-instrumentation/instrument/parser"
	"github.com/jattle/go-instrumentation/instrument/rewriter"
)

type foo struct {
	a int
}

func (f foo) echo() {
}

func (f *foo) zoo() {
}

// nolint
var tFunc = func(t *testing.T) {

}

func parseFunc(filename, functionname string) (fun *ast.FuncDecl, fset *token.FileSet) {
	var tFunc2 = func() {
		fmt.Println("tFunc2")
	}
	tFunc2()
	fset = token.NewFileSet()
	if file, err := parser.ParseFile(fset, filename, nil, 0); err == nil {
		for _, d := range file.Decls {
			if f, ok := d.(*ast.FuncDecl); ok && f.Name.Name == functionname {
				fun = f
				return
			}
		}
	}
	panic("function not found")
}

func printSelf(a int, b interface{}, aa func(), ctx context.Context, args ...any) (c int, d interface{}) {
	c = a
	d = b
	t := true
	ss := "xxx"
	funcArgs := []any{a, b, aa, ctx, args}
	fmt.Printf("a = %d, b = %v, ctx = %v, args = %v, t = %v, funcArgs = %v, ss = %v\n",
		a, b, ctx, args, t, funcArgs, ss)
	switch a {
	case 1:
		fmt.Println(1)
	case 2:
		fmt.Println(2)
	default:
		fmt.Println(0)
	}
	// Parse source file and extract the AST without comments for
	// this function, with position information referring to the
	// file set fset.
	funcAST, fset := parseFunc("example_test.go", "printSelf")
	ast.Print(fset, funcAST)

	// Print the function body into buffer buf.
	// The file set is provided to the printer so that it knows
	// about the original source formatting and can add additional
	// line breaks where they were present in the source.
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, funcAST.Body)

	// Remove braces {} enclosing the function body, unindent,
	// and trim leading and trailing white space.
	s := buf.String()
	s = s[1 : len(s)-1]
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n\t", "\n"))

	// Print the cleaned-up body text to stdout.
	fmt.Println(s)
	return
}

func ExampleFprint() {
	printSelf(1, string("abc"), func() {}, context.Background(), 1, 2, 3)

	// Output:
	// funcAST, fset := parseFunc("example_test.go", "printSelf")
	//
	// var buf bytes.Buffer
	// printer.Fprint(&buf, fset, funcAST.Body)
	//
	// s := buf.String()
	// s = s[1 : len(s)-1]
	// s = strings.TrimSpace(strings.ReplaceAll(s, "\n\t", "\n"))
	//
	// fmt.Println(s)
}

func TestModFunc(t *testing.T) {
	meta, err := iparser.ParseFile("example.go")
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = rewriter.RewritePatchASTFunc(meta)
	if err != nil {
		fmt.Println(err)
	}
	ast.Print(meta.FSet, meta.ASTFile)
	var buf bytes.Buffer
	printer.Fprint(&buf, meta.FSet, meta.ASTFile)

	// Remove braces {} enclosing the function body, unindent,
	// and trim leading and trailing white space.
	s := buf.String()
	s = s[1 : len(s)-1]
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n\t", "\n"))

	// Print the cleaned-up body text to stdout.
	fmt.Println(s)
}
