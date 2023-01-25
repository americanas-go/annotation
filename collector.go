package annotation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/americanas-go/errors"
	"golang.org/x/mod/modfile"
)

const (
	an = "@A"
	cb = "//"
)

func Collect(path string) ([]Block, error) {

	d, err := parseDir(path)
	if err != nil {
		return nil, err
	}

	return filterAnnotations(d)
}

func moduleName(basePath string, internalPath string) (string, error) {
	log.Tracef("getting module name")

	t := basePath + internalPath
	fileName := "go.mod"

	var dat []byte

	for {
		f := fmt.Sprintf("%v/%v", t, fileName)

		log.Debugf("attempt read go.mod on %s", f)

		dat, _ = os.ReadFile(f)
		if len(dat) > 0 {
			break
		}

		rel, _ := filepath.Rel(basePath, t)
		if rel == "." {
			break
		}

		t += "/.."
	}

	if len(dat) == 0 {
		return "", errors.NotFoundf("go.mod")
	}

	return modfile.ModulePath(dat), nil
}

func parseDir(rootPath string) (map[string]*ast.Package, error) {
	log.Tracef("parsing dir %s", rootPath)

	var err error

	var paths []string
	paths, err = scanDir(rootPath)
	if err != nil {
		return nil, err
	}

	pkgMap := make(map[string]*ast.Package)
	for _, path := range paths {
		fset := token.NewFileSet()

		var curPkgMap map[string]*ast.Package
		curPkgMap, err = parser.ParseDir(fset, path, nil, parser.ParseComments)

		if err != nil {
			return nil, err
		}

		for k, v := range curPkgMap {
			pkgMap[k] = v
		}
	}

	return pkgMap, nil
}

func scanDir(path string) ([]string, error) {
	log.Tracef("scan dir %s", path)

	var folders []string

	if err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {

		f, err = os.Stat(path)
		if err != nil {
			return err
		}

		mode := f.Mode()
		if mode.IsDir() {
			log.Debugf("found dir %s", path)
			folders = append(folders, path)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return folders, nil
}

func checkAnnotation(a string) bool {
	pre := strings.Join([]string{cb, an}, " ")
	if pre == a[0:len(pre)] {
		return true
	}
	return false
}

func filterAnnotations(d map[string]*ast.Package) (m []Block, err error) {

	log.Tracef("filtering annotations")

	var basePath string
	basePath, err = os.Getwd()
	if err != nil {
		return nil, err
	}

	for _, p := range d {
		log.Tracef("parsing package %s", p.Name)

		for k, f := range p.Files {
			log.Tracef("parsing file %s", k)

			tgt := strings.ReplaceAll(filepath.Dir(k), basePath, "")

			modName, err := moduleName(basePath, tgt)
			if err != nil {
				return nil, err
			}

			for _, g := range f.Comments {
				var contains bool
				var cmts []string
				for _, c := range g.List {
					if checkAnnotation(c.Text) {
						contains = true
						cmt := strings.ReplaceAll(c.Text,
							strings.Join([]string{cb, an, ""}, " "), "")
						cmts = append(cmts, cmt)
						log.Debugf("discovered annotation %s", cmt)
					}
				}
				if contains {

					w := strings.Split(strings.ReplaceAll(g.List[0].Text, cb, ""), " ")
					n := w[1]
					var title string
					if len(w) > 2 {
						title = strings.Join(w[2:], " ")
					}

					block := Block{
						File:    k,
						Path:    tgt,
						Module:  modName,
						Package: p.Name,
						Header: BlockHeader{
							Title:       title,
							Description: "",
						},
					}

					for name, obj := range f.Scope.Objects {
						if name == n {
							if obj.Kind.String() == "func" {
								block.Func = n
							} else {
								block.Struct = n
							}
							break
						}
					}

					for _, cmt := range cmts {
						fields := strings.Split(cmt, " ")
						block.Annotations = append(block.Annotations, Annotation{
							Name:  strings.ToLower(fields[0]),
							Value: strings.ToLower(strings.Join(fields[1:], " ")),
						})
					}

					m = append(m, block)
				}
			}
		}
	}
	return m, nil
}
