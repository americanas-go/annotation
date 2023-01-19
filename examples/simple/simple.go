package main

import (
	"context"
	"net/http"
)

// ExampleStruct Lorem ipsum dolor sit amet, consectetur adipiscing elit
// @A Package github.com/americanas-go/inject
// @A RelativePackage examples/simple
// @A App xpto
// @A HandlerType HTTP
// @A Type Interface
type ExampleStruct struct {
}

// FooStructMethod title
// @Grapper name=xpto
func (t *ExampleStruct) FooStructMethod(ctx context.Context, r *http.Request) (interface{}, error) {
	return Response{
		Message: "Hello world",
	}, nil
}

// New title
// @Inject context.Context
// @Provide *ExampleStruct name=xpto
func New(ctx context.Context) *ExampleStruct {
	return &ExampleStruct{}
}

// Xpto title
// @Inject *ExampleStruct name=xpto
// @Invoke
func Xpto(ex *ExampleStruct) {
}

// FooMethod Lorem ipsum dolor sit amet, consectetur adipiscing elit
// @A Package github.com/americanas-go/inject
// @A RelativePackage examples/simple
// @A App xpto
// @A HandlerType HTTP
// @A Type Function
// @A Path /foo
// @A Path /
// @A Method POST
// @A Consume application/json
// @A Consume application/yaml
// @A Produce application/json
// @A Param query foo bool true tiam sed efficitur purus
// @A Param query bar string true tiam sed efficitur purus
// @A Param path foo string tiam sed efficitur purus
// @A Param path bar string tiam sed efficitur purus
// @A Param header foo string true tiam sed efficitur purus
// @A Param header bar string true tiam sed efficitur purus
// @A Body github.com/americanas-go/inject/examples/simple.Request
// @A Response 201 github.com/americanas-go/inject/examples/simple.Response tiam sed efficitur purus, at lacinia magna
func FooMethod(ctx context.Context, r *http.Request) (interface{}, error) {
	return Response{
		Message: "Hello world",
	}, nil
}

type Response struct {
	Message string
}
