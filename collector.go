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
	entries      []Entry
}

func Collect(options ...Option) (*Collector, error) {
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
			return nil, errors.New("packages and path are required")
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
	return c, nil
}

func (c *Collector) Entries() []Entry {
	return c.entries
}

func (c *Collector) EntriesWithResultType(annotation string, result string) (entries []Entry) {
	for _, entry := range c.Entries() {
		if entry.IsStruct() {
			continue
		}
		var r, a bool
		for _, res := range entry.Func.Results {
			if res.Type == result {
				r = true
				break
			}
		}
		for _, ann := range entry.Annotations {
			if ann.Name == annotation {
				a = true
				break
			}
		}
		if a && r {
			entries = append(entries, entry)
		}
	}
	return entries
}

func (c *Collector) EntriesWith(annotation string) (entries []Entry) {
	for _, entry := range c.Entries() {
		for _, ann := range entry.Annotations {
			if ann.Name == annotation {
				entries = append(entries, entry)
				break
			}
		}
	}
	return entries
}

func (c *Collector) EntriesWithPrefix(prefix string) (entries []Entry) {
	for _, entry := range c.Entries() {
		for _, ann := range entry.Annotations {
			if strings.HasPrefix(ann.Name, prefix) {
				entries = append(entries, entry)
				break
			}
		}
	}
	return entries
}

func (c *Collector) filterPackages(value string) (err error) {

	if c.pkgProcessed[value] || !c.isAllowedPackage(value) {
		log.Debugf("the package %s has already been processed or is not in the list", value)
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

			log.Debugf("parsing import %s", imp)

			err := c.filterPackages(imp.String())
			if err != nil {
				return err
			}
		}

		entries, err := c.filterFiles(p)
		if err != nil {
			return err
		}

		c.entries = append(c.entries, entries...)
	}

	return nil
}

func (c *Collector) filterFiles(p *packages.Package) (m []Entry, err error) {
	for _, file := range p.Syntax {

		log.Tracef("parsing file %s", file.Name)

		var modName string
		if p.Module != nil {
			modName = p.Module.Path
		}

		entry := Entry{
			File:    file.Name.String(),
			Path:    p.PkgPath,
			Module:  modName,
			Package: p.Name,
		}

		for _, commentGroup := range file.Comments {

			entry, exists, err := c.processCommentGroups(commentGroup, p, file, entry)
			if err != nil {
				return nil, err
			}

			if exists {
				m = append(m, entry)
			}

		}
	}

	return m, nil
}

func (c *Collector) processCommentGroups(cg *ast.CommentGroup, pkg *packages.Package, file *ast.File, entry Entry) (Entry, bool, error) {

	log.Tracef("process comments %s", file.Name)

	eas, contains := c.getComments(cg)
	if !contains {
		return Entry{}, false, nil
	}

	var n string
	n, entry = c.parseHeader(cg, entry)

	for name, obj := range file.Scope.Objects {
		if name == n {
			if obj.Kind.String() == "func" {

				var err error
				entry, err = c.parseFunc(entry, pkg, name, file)
				if err != nil {
					return Entry{}, false, err
				}

			} else {
				entry.Struct = n
			}
			break
		}
	}

	entry.Annotations = eas

	return entry, true, nil
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
		log.Debugf("there is no annotation in the comment Entry")
		return nil, false
	}
	return cmts, true
}

func (c *Collector) extractAnnotation(cmt string) (Annotation, bool) {

	log.Tracef("extracting an annotation from the comment. %s", cmt)

	if !strings.HasPrefix(cmt, "// @") {
		log.Debugf("the comment is not an annotation. %s", cmt)
		return Annotation{}, false
	}

	if !c.isValidAnnotation(cmt) {
		log.Warnf("The annotation does not follow the format and will be ignored. %s", cmt)
		return Annotation{}, false
	}

	for _, filter := range c.filters {
		a := strings.Join([]string{"@", filter}, "")
		if !strings.Contains(cmt, a) {
			log.Warnf("The annotation is valid but will be ignored as it is not included in the filters. %s", cmt)
			continue
		}
		name, value := c.splitNameValue(cmt)
		log.Infof("discovered annotation %s with values (%s)", name, value)
		return NewAnnotation(name, value), true
	}
	return Annotation{}, false
}

func (c *Collector) splitNameValue(cmt string) (string, string) {
	fields := strings.Split(cmt, "(")
	names := strings.Split(fields[0], "@")
	name := strings.TrimSpace(names[1])
	value := strings.ReplaceAll(strings.TrimSpace(fields[1]), ")", "")
	return name, value
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

func (c *Collector) parseHeader(cg *ast.CommentGroup, Entry Entry) (string, Entry) {

	log.Tracef("parsing header on the comment group")

	w := strings.Split(strings.ReplaceAll(cg.List[0].Text, "//", ""), " ")
	n := w[1]
	var title string
	if len(w) > 2 {
		title = strings.Join(w[2:], " ")
	}

	Entry.Header.Title = title
	Entry.Header.Description = "// TODO"

	return n, Entry
}

func (c *Collector) parseFunc(Entry Entry, pkg *packages.Package, name string, file *ast.File) (Entry, error) {

	log.Tracef("parsing func %s on package %s", name, pkg.Name)

	Entry.Func = EntryFunc{
		Name: name,
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch fn := node.(type) {
		case *ast.FuncDecl:

			if fn.Name.Name == name {

				sig, _ := pkg.TypesInfo.ObjectOf(fn.Name).(*types.Func)
				if sig != nil {

					s := sig.Type().(*types.Signature)

					Entry = c.parseFuncParams(name, s, Entry)
					Entry = c.parseFuncResults(name, s, Entry)

				}

			}
		}
		return true
	})
	return Entry, nil
}

func (c *Collector) parseFuncParams(name string, s *types.Signature, Entry Entry) Entry {
	log.Tracef("parsing func %s params", name)

	params := s.Params()
	if params.Len() > 0 {
		for i := 0; i < params.Len(); i++ {
			param := params.At(i)
			Entry.Func.Parameters = append(Entry.Func.Parameters, EntryFuncParameter{
				Name: param.Name(),
				Type: param.Type().String(),
			})
		}
	}

	return Entry
}

func (c *Collector) parseFuncResults(name string, s *types.Signature, entry Entry) Entry {
	log.Tracef("parsing func %s results", name)
	results := s.Results()
	if results.Len() > 0 {
		for i := 0; i < results.Len(); i++ {
			entry.Func.Results = append(entry.Func.Results, EntryFuncResult{
				Type: results.At(i).Type().String(),
				Name: results.At(i).Name(),
			})
		}
	}
	return entry
}
