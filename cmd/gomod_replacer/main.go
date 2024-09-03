package main

import (
	"flag"
	"log"
	"os"
	"path"
	"regexp"

	"github.com/jattle/go-instrumentation/internal/gomodreplacer"
)

var (
	projectPath       = flag.String("project_path", "", "golang project path")
	pathPattern       = flag.String("path_pattern", "", "dep module path pattern")
	pkgmodReplacePath = flag.String("pkgmod_path", "instrumented_pkgmods", "dir to store instrumented dep pkgs")
)

func main() {
	flag.Parse()
	if *projectPath == "" {
		flag.Usage()
		return
	}
	var pathExpr *regexp.Regexp
	if *pathPattern != "" {
		pathExpr = regexp.MustCompile(*pathPattern)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("get wd failed, err: %+v\n", err)
	}
	_ = os.Chdir(*projectPath)
	modFilePath := path.Join(*projectPath, gomodreplacer.GoModFile)
	if modFilePath == gomodreplacer.GoModFile {
		modFilePath = path.Join(cwd, modFilePath)
	}
	// download deps
	descs, err := gomodreplacer.GenModuleDepDesc(*projectPath)
	if err != nil {
		log.Fatalf("download gomod deps failed, try again later. err: %+v", err)
		return
	}
	selectedDescs := gomodreplacer.SelectModuleDescs(descs, pathExpr)
	if err := gomodreplacer.CopyDeps(*pkgmodReplacePath, selectedDescs); err != nil {
		log.Fatalf("copy gomod deps to current dir failed: %+v", err)
	}
	if err := gomodreplacer.AddReplacesForGoMod(modFilePath, *pkgmodReplacePath, selectedDescs); err != nil {
		log.Fatalf("add replaces for project gomod failed, err: %+v", err)
	}
}
