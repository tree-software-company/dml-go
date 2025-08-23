package main

import (
	"log"

	"github.com/tree-software-company/dml-go/dml"
)

func main() {
	defaults := map[string]any{
		"server.name":    "MyApp1",
		"server.port":    8080,
		"server.timeout": 30,
		"database.host":  "localhost",
		"database.port":  5432,
	}

	err := dml.SetDefaultsToFile("testdata/example1.dml", defaults)
	if err != nil {
		log.Fatal(err)
	}
}
