package main

import (
	"log"

	"github.com/akrylysov/algnhsa"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/nametaginc/tvm"
)

func main() {
	svr, err := tvm.NewServer()
	if err != nil {
		log.Fatalf("cannot initialize server: %s", err)
	}

	handler := algnhsa.Handler(svr, &algnhsa.Options{
		RequestType: algnhsa.RequestTypeALB,

		// Note: you must include content-types with binary responses here otherwise
		// they will get cut off by lambda.
		BinaryContentTypes: []string{
			"*/*",
		},
	})

	lambda.StartHandler(handler)
}
