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
		annotation.WithPath(basePath+"/examples/method/app"),
		annotation.WithPackages("github.com/americanas-go/annotation"),
	)
	if err != nil {
		log.Error(err.Error())
	}

	j, _ := yaml.Marshal(collector.entries())
	fmt.Println(string(j))

}
