package main

import (
	"encoding/json"
	"fmt"
	"github.com/americanas-go/annotation"
	"github.com/americanas-go/log"
	"github.com/americanas-go/log/contrib/rs/zerolog.v1"
	"gopkg.in/yaml.v3"
	"os"
)

type RestResponse struct {
	Code        int    `attr:"code"`
	Type        string `attr:"type"`
	Description string `attr:"description"`
}

func main() {

	annotation.WithLogger(zerolog.NewLogger(zerolog.WithLevel("TRACE")))

	basePath, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}

	log.Infof("current path is %s", basePath)

	collector, err := annotation.Collect(
		annotation.WithFilters("Rest", "Boost", "Inject"),
		annotation.WithPath(basePath+"/examples/decode/app"),
		annotation.WithPackages("github.com/americanas-go/annotation"),
	)
	if err != nil {
		log.Error(err.Error())
	}

	blocks := collector.entries()

	j, _ := yaml.Marshal(blocks)
	fmt.Println(string(j))

	for _, block := range blocks {
		for _, annon := range block.Annotations {
			if annon.Name == "RestResponse" {
				rp := RestResponse{}
				if err := annon.Decode(&rp); err != nil {
					panic(err)
				}

				j, _ := json.Marshal(rp)
				fmt.Println(string(j))
			}
		}
	}

}
