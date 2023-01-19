package annotation

/*
import (
	"strings"

	ustrings "github.com/americanas-go/utils/strings"

	"github.com/ettle/strcase"
)

type Annotations map[string][]string

func (m *Annotations) extractParamsWithTypes(a []string, tp ...AnnotationType) (as []Annotation, err error) {

	var astr []string
	for _, v := range tp {
		astr = append(astr, v.String())
	}

	for _, v := range a {
		an := m.extractAnn(v)
		if ustrings.SliceContains(astr, an) {
			at, err := ParseAnnotationType(an)
			if err != nil {
				log.Errorf("error on parse annotation %s. %s", an, err.Error())
				continue
			}
			as = append(as, Annotation{
				Type:  at,
				Value: v,
			})
		}
	}

	return as, nil
}

func (m *Annotations) extractAnn(a string) string {
	return strcase.ToCase(strings.Split(a, " ")[0], strcase.UpperCase, '_')
}
*/
