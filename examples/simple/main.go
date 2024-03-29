package main

import (
	"fmt"
	"github.com/americanas-go/annotation"
	"github.com/americanas-go/log"
	"github.com/americanas-go/log/contrib/rs/zerolog.v1"
	"gopkg.in/yaml.v3"
	"os"
)

func main() {

	annotation.WithLogger(zerolog.NewLogger(zerolog.WithLevel("TRACE")))

	basePath, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}

	log.Infof("current path is %s", basePath)

	collector, err := annotation.Collect(
		// annotation.WithFilters("Inject"),
		annotation.WithPath(basePath),
		annotation.WithPackages("github.com/americanas-go/annotation", "github.com/jpfaria/tests/annotated"),
	)
	if err != nil {
		log.Error(err.Error())
	}

	j1, _ := yaml.Marshal(collector.Entries())
	fmt.Println(string(j1))

	entries1 := collector.EntriesWith("MyMethodAnnotation")
	j2, _ := yaml.Marshal(entries1)
	fmt.Println(string(j2))

}
