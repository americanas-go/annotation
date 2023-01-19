package annotation

type BlockHeader struct {
	Title       string
	Description string
}

type Block struct {
	Header      BlockHeader
	File        string
	Package     string
	Func        string
	Struct      string
	Annotations map[string]Annotation
}

func fromMap([]string) Block {
	return Block{}
}
