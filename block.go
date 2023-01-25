package annotation

type BlockHeader struct {
	Title       string
	Description string
}

type Block struct {
	Header      BlockHeader
	Module      string
	File        string
	Path        string
	Package     string
	Func        string
	Struct      string
	Annotations []Annotation
}

func (b *Block) IsStruct() bool {
	return b.Struct != "" && b.Func == ""
}

func (b *Block) IsFunc() bool {
	return b.Struct == "" && b.Func != ""
}
