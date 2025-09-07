package main

import (
	"log"

	"github.com/tree-software-company/dml-go/dml"
)

func main() {
	err := dml.SetDefaultsToFile("testdata/example1.dml", map[string]any{
		"server.port":    "test5",
		"server.timeout": "test2",
		"name":"MyApp",
	}, false) 

	if err != nil {
		log.Fatal("❌ Failed to apply defaults:", err)
	}

	cfg, err := dml.NewConfig("testdata/example.dml")

	if err != nil {
		log.Fatal(err)
	}
}
