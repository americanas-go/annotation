package annotation

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	an = "@A"
	cb = "//"
)

var (
	corePkgs = []string{
		"golang.org",
		"archive",
		"zip",
		"bufio",
		"builtin",
		"bytes",
		"compress",
		"container",
		"context",
		"crypto",
		"database",
		"debug",
		"embed",
		"encoding",
		"errors",
		"expvar",
		"flag",
		"fmt",
		"go",
		"hash",
		"html",
		"image",
		"index",
		"io",
		"log",
		"math",
		"mime",
		"net",
		"os",
		"path",
		"plugin",
		"reflect",
		"regexp",
		"runtime",
		"sort",
		"strconv",
		"strings",
		"sync",
		"syscall",
		"testing",
		"text",
		"time",
		"unicode",
		"unsafe",
	}
)

func Collect(path string) ([]Block, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypesInfo | packages.NeedSyntax |
			packages.NeedFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes |
			packages.NeedEmbedFiles | packages.NeedExportFile | packages.NeedModule | packages.NeedCompiledGoFiles,
	}
	processed := make(map[string]bool)
	return filterPackages(cfg, path, processed)
}

func checkAnnotation(a string) bool {
	pre := strings.Join([]string{cb, an}, " ")
	if len(a) > len(pre) && pre == a[0:len(pre)] {
		return true
	}
	return false
}

func isCorePackage(pkgPath string) bool {
	for _, n := range corePkgs {
		if strings.HasPrefix(pkgPath, n) {
			return true
		}
	}
	return false
}

func filterPackages(cfg *packages.Config, value string, processed map[string]bool) (m []Block, err error) {

	if processed[value] || isCorePackage(value) {
		return
	}

	log.Tracef("filtering packages on %s", value)

	pkgs, err := packages.Load(cfg, value)
	if err != nil {
		return nil, err
	}

	processed[value] = true

	for _, p := range pkgs {
		log.Tracef("parsing package %s", p.String())
		for _, imp := range p.Imports {
			mm, err := filterPackages(cfg, imp.String(), processed)
			if err != nil {
				return nil, err
			}
			m = append(m, mm...)
		}
		mm, err := filterFiles(p)
		if err != nil {
			return nil, err
		}
		m = append(m, mm...)
	}

	return m, nil
}

func filterFiles(p *packages.Package) (m []Block, err error) {
	for _, file := range p.Syntax {

		log.Tracef("parsing file %s", file.Name.String())

		var modName string
		if p.Module != nil {
			modName = p.Module.Path
		}

		block := Block{
			File:    file.Name.String(),
			Path:    p.PkgPath,
			Module:  modName,
			Package: p.Name,
		}

		for _, commentGroup := range file.Comments {

			block, exists, err := processCommentGroups(commentGroup, p, file, block)
			if err != nil {
				return nil, err
			}

			if exists {
				m = append(m, block)
			}

		}
	}

	return m, nil
}

func getComments(cg *ast.CommentGroup) ([]string, bool) {
	var contains bool
	var cmts []string
	for _, c := range cg.List {
		if !checkAnnotation(c.Text) {
			continue
		}
		contains = true
		cmt := strings.ReplaceAll(c.Text,
			strings.Join([]string{cb, an, ""}, " "), "")
		cmts = append(cmts, cmt)
		log.Debugf("discovered annotation %s", cmt)
	}
	if !contains {
		return nil, false
	}
	return cmts, true
}

func parseHeader(cg *ast.CommentGroup, block Block) (string, Block) {
	w := strings.Split(strings.ReplaceAll(cg.List[0].Text, cb, ""), " ")
	n := w[1]
	var title string
	if len(w) > 2 {
		title = strings.Join(w[2:], " ")
	}

	block.Header.Title = title
	block.Header.Description = ""

	return n, block
}

func processCommentGroups(cg *ast.CommentGroup, pkg *packages.Package, file *ast.File, block Block) (Block, bool, error) {
	cmts, contains := getComments(cg)
	if !contains {
		return Block{}, false, nil
	}

	var n string
	n, block = parseHeader(cg, block)

	for name, obj := range file.Scope.Objects {
		if name == n {
			if obj.Kind.String() == "func" {

				var err error
				block, err = parseFunc(block, pkg, name, file)
				if err != nil {
					return Block{}, false, err
				}

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

	return block, true, nil
}

func parseFunc(block Block, pkg *packages.Package, n string, file *ast.File) (Block, error) {

	block.Func = BlockFunc{
		Name: n,
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch fn := node.(type) {
		case *ast.FuncDecl:

			if fn.Name.Name == n {

				sig, _ := pkg.TypesInfo.ObjectOf(fn.Name).(*types.Func)
				if sig != nil {

					s := sig.Type().(*types.Signature)

					block = parseFuncParams(s, block)
					block = parseFuncResults(s, block)

				}

			}
		}
		return true
	})
	return block, nil
}

func parseFuncParams(s *types.Signature, block Block) Block {
	params := s.Params()
	if params.Len() > 0 {
		for i := 0; i < params.Len(); i++ {
			param := params.At(i)
			block.Func.Parameters = append(block.Func.Parameters, BlockFuncParameter{
				Name: param.Name(),
				Type: param.Type().String(),
			})
		}
	}

	return block
}

func parseFuncResults(s *types.Signature, block Block) Block {
	results := s.Results()
	if results.Len() > 0 {
		for i := 0; i < results.Len(); i++ {
			block.Func.Results = append(block.Func.Results, BlockFuncResult{
				Type: results.At(i).Type().String(),
				Name: results.At(i).Name(),
			})
		}
	}
	return block
}
