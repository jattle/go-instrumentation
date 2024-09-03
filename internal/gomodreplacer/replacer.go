package gomodreplacer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
)

const (
	GoModFile = "go.mod"
)

// ModuleCacheDesc result of go mod download
type ModuleCacheDesc struct {
	Path     string // module path
	Version  string // module version
	Error    string // error loading module
	Info     string // absolute path to cached .info file
	GoMod    string // absolute path to cached .mod file
	Zip      string // absolute path to cached .zip file
	Dir      string // absolute path to cached source root directory
	Sum      string // checksum for path, version (as in go.sum)
	GoModSum string // checksum for go.mod (as in go.sum)
}

// GenModuleDepDesc return all module dependences for project
func GenModuleDepDesc(projectPath string) (descs []ModuleCacheDesc, err error) {
	cmd := exec.Command("go", "mod", "download", "-json")
	cmd.Dir = projectPath
	stdOut := bytes.Buffer{}
	stdErr := bytes.Buffer{}
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("gomod analysis failed, err: %+v, stderr: %s", err, stdErr.String())
		return
	}
	out := stdOut.String()
	dec := json.NewDecoder(bytes.NewBufferString(out))
	var desc ModuleCacheDesc
	for err = dec.Decode(&desc); err == nil; err = dec.Decode(&desc) {
		descs = append(descs, desc)
	}
	if err == io.EOF {
		err = nil
	}
	return
}

// ParseModFile parse mod file and return file info
func ParseModFile(filename string) (fi *modfile.File, err error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	fi, err = modfile.Parse(filename, data, nil)
	if err != nil {
		return
	}
	if fi.Go == nil || fi.Module == nil {
		err = fmt.Errorf("filename: %s, invalid fi: %+v", filename, fi)
		return
	}
	return
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if !(dfi.Mode().IsRegular()) {
		return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
	}
	if os.SameFile(sfi, dfi) {
		return
	}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

// CopyDir copy files from src dir to dst dir
func CopyDir(src, dst string) error {
	// assume dst dir is empty
	fs, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("traverse dir %s failed, err: %+v", src, err)
	}
	for _, v := range fs {
		if v.IsDir() {
			srcSubDir := path.Join(src, v.Name())
			dstSubDir := path.Join(dst, v.Name())
			if err := os.Mkdir(dstSubDir, 0775); err != nil && !os.IsExist(err) {
				return err
			}
			if err := CopyDir(srcSubDir, dstSubDir); err != nil {
				return err
			}
		} else if v.Type().IsRegular() {
			srcFile := path.Join(src, v.Name())
			dstFile := path.Join(dst, v.Name())
			if err := copyFileContents(srcFile, dstFile); err != nil {
				return err
			}
		}
	}
	return nil
}

// SelectModuleDescs select matched modules
func SelectModuleDescs(descs []ModuleCacheDesc, pathPat *regexp.Regexp) []ModuleCacheDesc {
	selectedDescs := make([]ModuleCacheDesc, 0)
	// select deps
	for _, desc := range descs {
		gomod := path.Join(desc.Dir, GoModFile)
		// ignore module which mssing gomod file
		if _, err := os.Stat(gomod); os.IsNotExist(err) {
			continue
		}
		if pathPat != nil && pathPat.MatchString(desc.Path) {
			selectedDescs = append(selectedDescs, desc)
		}
	}
	if len(selectedDescs) == 0 {
		selectedDescs = descs
	}
	return selectedDescs
}

func getPkgStoreDir(pkgModDir, dir string) string {
	pkgDir := dir
	const modPat = "/pkg/mod/"
	idx := strings.Index(pkgDir, modPat)
	if idx != -1 {
		pkgDir = pkgDir[idx+len(modPat):]
	}
	pkgDir = path.Join(pkgModDir, pkgDir)
	return pkgDir
}

// CopyDeps copy dep module dirs to pkgModDir
func CopyDeps(pkgModDir string, descs []ModuleCacheDesc) error {
	if err := os.MkdirAll(pkgModDir, 0775); err != nil {
		return err
	}
	for _, desc := range descs {
		pkgDir := getPkgStoreDir(pkgModDir, desc.Dir)
		if err := os.MkdirAll(pkgDir, 0775); err != nil {
			log.Fatalf("mkdir %s failed. err: %+v", pkgDir, err)
		}
		if err := CopyDir(desc.Dir, pkgDir); err != nil {
			log.Fatalf("copy dir from %s to %s failed. err: %+v", desc.Dir, pkgDir, err)
		}
	}
	return nil
}

func emitReplaces(pkgModDir string, descs []ModuleCacheDesc, modFile *modfile.File) []string {
	replaces := make([]string, 0, len(descs))
	for _, desc := range descs {
		pkgDir := getPkgStoreDir(pkgModDir, desc.Dir)
		if idx := slices.IndexFunc(modFile.Replace, func(r *modfile.Replace) bool {
			return r.Old.Path == desc.Path && r.New.Path == pkgDir
		}); idx == -1 {
			replaces = append(replaces, fmt.Sprintf("replace %s => %s\n", desc.Path, pkgDir))
		}
	}
	return replaces
}

func rewriteGoMod(path string, replaces []string) error {
	// rewrite go mod
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, r := range replaces {
		f.WriteString(r)
	}
	return nil
}

// AddReplacesForGoMod add necessary replaces for project go mod file
func AddReplacesForGoMod(modFilePath, pkgModDir string, descs []ModuleCacheDesc) error {
	modFile, err := ParseModFile(modFilePath)
	if err != nil {
		return fmt.Errorf("parse modfile failed, file: %s, err: %+v", modFilePath, err)
	}
	replaces := emitReplaces(pkgModDir, descs, modFile)
	if len(replaces) == 0 {
		return nil
	}
	// rewrite go mod
	if err := rewriteGoMod(modFilePath, replaces); err != nil {
		return fmt.Errorf("rewrite gomod failed, err: %+v", err)
	}
	return nil
}
