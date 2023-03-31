package annotation

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
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

	pkgMap, err := parseDir(path)
	if err != nil {
		return nil, err
	}

	return filterPackages(pkgMap)
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

type Pkg struct {
	FSet *token.FileSet
	Pkg  *ast.Package
}

func parseDir(rootPath string) (map[string]Pkg, error) {
	log.Tracef("parsing dir %s", rootPath)

	var err error

	var paths []string
	paths, err = scanDir(rootPath)
	if err != nil {
		return nil, err
	}

	pkgMap := make(map[string]Pkg)
	for _, path := range paths {
		fset := token.NewFileSet()

		var curPkgMap map[string]*ast.Package
		curPkgMap, err = parser.ParseDir(fset, path, nil, parser.ParseComments)

		if err != nil {
			return nil, err
		}

		for k, v := range curPkgMap {
			pkgMap[k] = Pkg{
				FSet: fset,
				Pkg:  v,
			}
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
	if len(a) > len(pre) && pre == a[0:len(pre)] {
		return true
	}
	return false
}

func filterPackages(pkgMap map[string]Pkg) (m []Block, err error) {

	log.Tracef("filtering annotations")

	var basePath string
	basePath, err = os.Getwd()
	if err != nil {
		return nil, err
	}

	for _, p := range pkgMap {
		log.Tracef("parsing package %s", p.Pkg.Name)
		mm, err := filterFiles(p, basePath)
		if err != nil {
			return nil, err
		}
		m = append(m, mm...)
	}
	return m, nil
}

func filterFiles(p Pkg, basePath string) (m []Block, err error) {
	for fileName, file := range p.Pkg.Files {
		log.Tracef("parsing file %s", fileName)

		tgt := strings.ReplaceAll(filepath.Dir(fileName), basePath, "")

		modName, err := moduleName(basePath, tgt)
		if err != nil {
			return nil, err
		}

		block := Block{
			File:    fileName,
			Path:    tgt,
			Module:  modName,
			Package: p.Pkg.Name,
		}

		for _, commentGroup := range file.Comments {

			block, exists, err := processCommentGroups(commentGroup, file, p.FSet, block)
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

func processCommentGroups(cg *ast.CommentGroup, file *ast.File, fset *token.FileSet, block Block) (Block, bool, error) {
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
				block, err = parseFunc(block, n, fset, file)
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

func parseFunc(block Block, n string, fset *token.FileSet, file *ast.File) (Block, error) {

	block.Func = BlockFunc{
		Name: n,
	}

	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("", fset, []*ast.File{file}, nil)
	if err != nil {
		return Block{}, err
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch fn := node.(type) {
		case *ast.FuncDecl:

			if fn.Name.Name == n {

				sig, _ := pkg.Scope().Lookup(fn.Name.Name).(*types.Func)
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
