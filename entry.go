package annotation

type EntryHeader struct {
	Title       string
	Description string
}

type Entry struct {
	Header      EntryHeader
	Module      string
	File        string
	Path        string
	Package     string
	Func        EntryFunc
	Struct      string
	Annotations []Annotation
}

type EntryFunc struct {
	Name       string
	Parameters []EntryFuncParameter
	Results    []EntryFuncResult
}

type EntryFuncParameter struct {
	Name string
	Type string
}

type EntryFuncResult struct {
	Name string
	Type string
}

func (b *Entry) IsStruct() bool {
	return b.Struct != "" && b.Func.Name == ""
}

func (b *Entry) IsFunc() bool {
	return b.Struct == "" && b.Func.Name != ""
}

func (b *Entry) IsMethod() bool {
	return b.Struct != "" && b.Func.Name != ""
}
