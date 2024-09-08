package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

// FileMeta file meta
type FileMeta struct {
	FileName string
	FSet     *token.FileSet
	ASTFile  *ast.File
	Content  []byte
}

// ParseFile parse go source file
func ParseFile(filename string) (meta FileMeta, err error) {
	meta.FileName = filename
	meta.FSet = token.NewFileSet()
	if meta.ASTFile, err = parser.ParseFile(meta.FSet, filename, nil, parser.ParseComments); err != nil {
		err = fmt.Errorf("parse file %s failed: %w", filename, err)
		return
	}
	if meta.Content, err = os.ReadFile(filename); err != nil {
		err = fmt.Errorf("read file %s failed: %w", filename, err)
	}
	return
}
