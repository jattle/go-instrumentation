package rewriter

import (
	"bytes"
	"fmt"
	"sort"
)

// EditType edit type
type EditType int

const (
	EditTypeAdd EditType = iota + 1
	EditTypeDel
	EditTypeReplace
)

// Edit edition for original binary content
type Edit struct {
	OpType           EditType
	BeginPos, EndPos int
	Content          []byte // for add or replace op
}

// EditSlice edit slice
type EditSlice []Edit

var _ sort.Interface = (EditSlice)(nil)

// Len slice length
func (e EditSlice) Len() int {
	return len(e)
}

// Less less than
func (e EditSlice) Less(i, j int) bool {
	if e[i].BeginPos == e[j].BeginPos {
		return e[i].EndPos < e[j].EndPos
	}
	return e[i].BeginPos < e[j].BeginPos
}

// Swap swap slice elements
func (e EditSlice) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

// FileRewriter apply edits and rewrite source file content
type FileRewriter struct {
	Content []byte // original content
	Edits   []Edit
}

// Rewrite rewrite file
func (f *FileRewriter) Rewrite() (content []byte, err error) {
	var buf bytes.Buffer
	lastPos := 0
	lastEditPos := -1
	for _, e := range f.Edits {
		if lastEditPos != e.BeginPos {
			buf.Write(f.Content[lastPos:e.BeginPos])
		}
		switch e.OpType {
		case EditTypeAdd:
			buf.Write(e.Content)
			lastPos = e.EndPos
		case EditTypeDel:
			// just ignore
			lastPos = e.EndPos + 1
		case EditTypeReplace:
			buf.Write(e.Content)
			lastPos = e.EndPos + 1
		default:
			err = fmt.Errorf("unsupported edit type %+v", e.OpType)
			return
		}
		lastEditPos = e.BeginPos
	}
	buf.Write(f.Content[lastPos:])
	content = buf.Bytes()
	return
}
