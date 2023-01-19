package annotation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

const (
	an = "@A"
	cb = "//"
)

func CollectByPath(path string) ([]Block, error) {
	d, err := parseDir(path)
	if err != nil {
		return nil, err
	}

	return filterAnnotations(d)
}

func parseDir(rootPath string) (map[string]*ast.Package, error) {
	log.Tracef("parsing dir %s", rootPath)

	var err error

	var paths []string
	paths, err = scanDir(rootPath, []string{})
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

func scanDir(path string, ignore []string) ([]string, error) {
	log.Tracef("scab dir %s", path)

	var folders []string

	if err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {

		_continue := false

		for _, i := range ignore {

			if strings.Index(path, i) != -1 {
				_continue = true
			}
		}

		if _continue == false {

			f, err = os.Stat(path)
			if err != nil {
				log.Fatal(err)
			}

			mode := f.Mode()
			if mode.IsDir() {
				log.Debugf("found dir %s", path)
				folders = append(folders, path)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return folders, nil
}

func filterAnnotations(d map[string]*ast.Package) (m []Block, err error) {

	log.Tracef("filtering annotations")

	for _, p := range d {
		log.Tracef("parsing package %s", p.Name)
		for k, f := range p.Files {
			log.Tracef("parsing file %s", k)

			for _, g := range f.Comments {
				var contains bool
				var cmts []string
				for _, c := range g.List {
					pre := strings.Join([]string{cb, an}, " ")
					if pre == c.Text[0:len(pre)] {
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

					block := Block{
						File:    k,
						Package: p.Name,
						Header: BlockHeader{
							Title:       "",
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

					m = append(m, block)
				}
			}
		}
	}
	return m, nil
}
