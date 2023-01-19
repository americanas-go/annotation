package annotation

import "strings"

type Annotation struct {
	Name  string
	Value string
}

func (m *Annotation) SimpleValue() string {
	return strings.Split(m.Value, " ")[1]
}
