package annotation

import (
	"errors"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Option func(*Collector) error

func WithFilters(filters ...string) Option {
	return func(c *Collector) error {
		if filters == nil {
			return errors.New("no types informed")
		}
		c.filters = filters
		return nil
	}
}

func WithPackages(pkgs ...string) Option {
	return func(c *Collector) error {
		if pkgs == nil {
			return errors.New("no packages informed")
		}
		for _, pkg := range pkgs {
			c.pkgs = append(c.pkgs, pkg)
		}
		return nil
	}
}

func WithPath(path string) Option {
	return func(c *Collector) error {
		if path == "" {
			return errors.New("no path informed")
		}
		c.path = path
		return nil
	}
}

type Collector struct {
	filters      []string
	pkgs         []string
	path         string
	pkgProcessed map[string]bool
	pkgConfig    *packages.Config
	m            []Block
}

func Collect(options ...Option) ([]Block, error) {
	c := &Collector{
		pkgProcessed: make(map[string]bool),
		pkgConfig: &packages.Config{
			Mode: packages.NeedName | packages.NeedTypesInfo | packages.NeedSyntax |
				packages.NeedFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes |
				packages.NeedEmbedFiles | packages.NeedExportFile | packages.NeedModule | packages.NeedCompiledGoFiles,
		},
	}
	for _, opt := range options {
		err := opt(c)
		if err != nil {
			panic(err.Error())
		}
	}

	if len(c.filters) == 0 {
		c.filters = []string{""}
	}

	if c.pkgs == nil || c.path == "" {
		return nil, errors.New("packages and path are required")
	}
	log.Tracef("starting to collect annotations. filters: %v packages: %v path: %s", c.filters, c.pkgs, c.path)
	if err := c.filterPackages(c.path); err != nil {
		return nil, err
	}
	return c.m, nil
}

func (c *Collector) filterPackages(value string) (err error) {

	if c.pkgProcessed[value] || !c.isAllowedPackage(value) {
		return
	}

	log.Tracef("filtering packages on %s", value)

	pkgs, err := packages.Load(c.pkgConfig, value)
	if err != nil {
		return err
	}

	c.pkgProcessed[value] = true

	for _, p := range pkgs {

		log.Tracef("parsing package %s", p.String())

		for _, imp := range p.Imports {
			err := c.filterPackages(imp.String())
			if err != nil {
				return err
			}
		}

		mm, err := c.filterFiles(p)
		if err != nil {
			return err
		}

		c.m = append(c.m, mm...)
	}

	return nil
}

func (c *Collector) filterFiles(p *packages.Package) (m []Block, err error) {
	for _, file := range p.Syntax {

		log.Tracef("parsing file %s", file.Name)

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

			block, exists, err := c.processCommentGroups(commentGroup, p, file, block)
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

func (c *Collector) processCommentGroups(cg *ast.CommentGroup, pkg *packages.Package, file *ast.File, block Block) (Block, bool, error) {

	log.Tracef("process comments %s", file.Name)

	eas, contains := c.getComments(cg)
	if !contains {
		return Block{}, false, nil
	}

	var n string
	n, block = c.parseHeader(cg, block)

	for name, obj := range file.Scope.Objects {
		if name == n {
			if obj.Kind.String() == "func" {

				var err error
				block, err = c.parseFunc(block, pkg, name, file)
				if err != nil {
					return Block{}, false, err
				}

			} else {
				block.Struct = n
			}
			break
		}
	}

	block.Annotations = eas

	return block, true, nil
}

func (c *Collector) getComments(cg *ast.CommentGroup) (cmts []Annotation, ok bool) {

	log.Tracef("get comments comments")

	var contains bool
	for _, cc := range cg.List {
		ea, ok := c.extractAnnotation(cc.Text)
		if !ok {
			continue
		}
		contains = true
		cmts = append(cmts, ea)
	}
	if !contains {
		log.Debugf("there is no annotation in the comment block")
		return nil, false
	}
	return cmts, true
}

func (c *Collector) extractAnnotation(expr string) (Annotation, bool) {

	log.Tracef("extracting an annotation from the comment. %s", expr)

	if !strings.HasPrefix(expr, "// @") {
		log.Debugf("the comment is not an annotation. %s", expr)
		return Annotation{}, false
	}

	if !c.isValidAnnotation(expr) {
		log.Warnf("The annotation does not follow the format and will be ignored. %s", expr)
		return Annotation{}, false
	}

	for _, filter := range c.filters {
		annon := strings.Join([]string{"@", filter}, "")
		if !strings.Contains(expr, annon) {
			log.Warnf("The annotation is valid but will be ignored as it is not included in the filters. %s", expr)
			continue
		}
		values := strings.Split(expr, "(")
		value := strings.ReplaceAll(values[1], ")", "")
		names := strings.Split(values[0], "@")
		name := strings.TrimSpace(names[1])
		log.Debugf("discovered annotation %s with values (%s)", name, value)
		return NewAnnotation(name, value), true
	}
	return Annotation{}, false
}

func (c *Collector) isValidAnnotation(input string) bool {

	log.Tracef("checking if it is a valid annotation. %s", input)

	split := strings.SplitN(input, "@", 2)

	// Checking if string starts with "// @"
	if !(len(split) == 2 && split[0] == "// ") {
		return false
	}

	// Checking if there are parentheses and if they have valid key-value pairs
	parenSplit := strings.SplitN(split[1], "(", 2)
	if len(parenSplit) == 2 {
		parenContent := strings.Trim(parenSplit[1], " )")
		pairs := strings.Split(parenContent, ",")
		for _, pair := range pairs {
			if !strings.Contains(pair, "=") {
				return false
			}
		}
		log.Debugf("there is a valid annotation in the comment")
		return true
	}

	return false
}

func (c *Collector) isAllowedPackage(pkgPath string) bool {
	for _, n := range c.pkgs {
		if strings.Contains(pkgPath, n) {
			return true
		}
	}
	return false
}

func (c *Collector) parseHeader(cg *ast.CommentGroup, block Block) (string, Block) {

	log.Tracef("parsing header on the comment group")

	w := strings.Split(strings.ReplaceAll(cg.List[0].Text, "//", ""), " ")
	n := w[1]
	var title string
	if len(w) > 2 {
		title = strings.Join(w[2:], " ")
	}

	block.Header.Title = title
	block.Header.Description = "// TODO"

	return n, block
}

func (c *Collector) parseFunc(block Block, pkg *packages.Package, name string, file *ast.File) (Block, error) {

	log.Tracef("parsing func %s on package %s", name, pkg.Name)

	block.Func = BlockFunc{
		Name: name,
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch fn := node.(type) {
		case *ast.FuncDecl:

			if fn.Name.Name == name {

				sig, _ := pkg.TypesInfo.ObjectOf(fn.Name).(*types.Func)
				if sig != nil {

					s := sig.Type().(*types.Signature)

					block = c.parseFuncParams(s, block)
					block = c.parseFuncResults(s, block)

				}

			}
		}
		return true
	})
	return block, nil
}

func (c *Collector) parseFuncParams(s *types.Signature, block Block) Block {
	log.Tracef("parsing func params")

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

func (c *Collector) parseFuncResults(s *types.Signature, block Block) Block {
	log.Tracef("parsing func results")
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
