package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/jattle/go-instrumentation/instrument"
)

var (
	source          = flag.String("source", "", "go source file to instrument")
	output          = flag.String("output", "", "output file to store instrumentation result")
	replace         = flag.Bool("replace", false, "replace source file with instrumentation result")
	patches         = flag.String("patches", "", "patch file separated by ,")
	funcExcludeExpr = flag.String("exclude_func_expr", "", "regex pattern of function to exclude from instrumentation")
)

func usage() {
	txt := `
	Usage: tool -source=[source filename] -output=[optional] -replace[optional] -patches=[patch file list]
	            -exclude_func_expr=[optional]
		   must provide source and patches option, if replace is provided, source file content will be overwritten,
		   otherwise output filename should be provided 
	`
	fmt.Fprintf(os.Stderr, "%s\n\n", txt)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if *source == "" || *patches == "" || (*output == "" && !*replace) {
		flag.Usage()
		flag.PrintDefaults()
		return
	}
	if *funcExcludeExpr != "" {
		instrument.FuncNameExcludeExpr = regexp.MustCompile(*funcExcludeExpr)
	}
	// parse source file
	sourceMeta, err := instrument.ParseFile(*source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse source %s failed, err: %+v\n", *source, err)
		return
	}
	// parse patches
	patchFiles := strings.Split(*patches, ",")
	patchMetas := make([]instrument.FileMeta, 0, len(patchFiles))
	for _, f := range patchFiles {
		meta, err := instrument.ParseFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse patch %s, failed, err: %+v\n", f, err)
			continue
		}
		patchMetas = append(patchMetas, meta)
	}
	defer func() {
		if e := recover(); e != nil {
			buf := [1024]byte{}
			sbuf := buf[:runtime.Stack(buf[:], false)]
			fmt.Fprintf(os.Stderr, "auto instrumentation exec failed, file: %s, err: %+v, stack: %s\n",
				*source, e, string(sbuf))
		}
	}()
	if err = instrument.RewriteSourceFile(sourceMeta, patchMetas); err != nil {
		fmt.Fprintf(os.Stderr, "rewrite source %s failed, err: %+v\n", *source, err)
		return
	}
	if *replace {
		*output = *source
	}
	if err = saveInstrmentation(sourceMeta, *output); err != nil {
		fmt.Fprintf(os.Stderr, "save instrumentation failed, err: %+v\n", err)
		return
	}
}

func saveInstrmentation(meta instrument.FileMeta, filename string) error {
	s, err := instrument.ASTToString(meta)
	if err != nil {
		return fmt.Errorf("convert file %s ast failed: %w", filename, err)
	}
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("open file %s failed: %w", filename, err)
	}
	defer f.Close()
	if n, err := f.WriteString(s); err != nil || n != len(s) {
		return fmt.Errorf("write file %s failed: write size %d, expected size: %d, err: %w",
			filename, n, len(s), err)
	}
	return nil
}
