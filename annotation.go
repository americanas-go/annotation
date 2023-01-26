package annotation

import (
	"strings"

	"github.com/mitchellh/mapstructure"
)

type Annotation struct {
	Name  string
	Value string
}

func (m *Annotation) RawValue() string {
	return m.Value
}

func (m *Annotation) AsValuedMap(fields []string) map[string]string {
	vl := strings.Split(m.Value, " ")
	mp := make(map[string]string)
	for i, v := range vl {
		mp[fields[i]] = v
	}
	return mp
}

func (m *Annotation) AsMap() map[string]string {
	entries := strings.Split(m.Value, " ")
	mp := make(map[string]string)
	for _, entry := range entries {
		e := strings.Split(entry, "=")
		if len(e) == 2 {
			mp[e[0]] = e[1]
		}
	}
	return mp
}

func (m *Annotation) AsStruct(o interface{}) error {
	mp := m.AsMap()
	return mapstructure.Decode(mp, o)
}

func (m *Annotation) AsValuedStruct(fields []string, o interface{}) error {
	mp := m.AsValuedMap(fields)
	return mapstructure.Decode(mp, o)
}
