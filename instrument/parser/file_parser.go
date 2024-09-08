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
	var content []byte
	if content, err = os.ReadFile(filename); err != nil {
		err = fmt.Errorf("read file %s failed: %w", filename, err)
		return
	}
	meta, err = ParseContent(filename, content)
	return
}

// ParseContent parse go source content
func ParseContent(filename string, content []byte) (meta FileMeta, err error) {
	meta.FileName = filename
	meta.Content = content
	meta.FSet = token.NewFileSet()
	if meta.ASTFile, err = parser.ParseFile(meta.FSet, filename, content, parser.ParseComments); err != nil {
		err = fmt.Errorf("parse file %s failed: %w", filename, err)
		return
	}
	return
}
