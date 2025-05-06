package main

import (
	"fmt"
	"log"

	"github.com/tree-software-company/dml-go/dml"
)

func main() {
	defaults := map[string]any{
		"server.port":    8080,
		"server.timeout": 15,
		"server.name":    "MyApp1",
		"database.host":  "localhost",
		"database.port":  5432,
	}

	err := dml.SetDefaultsToFile("testdata/example.dml", defaults)
	if err != nil {
		log.Fatal("❌ Failed to apply defaults:", err)
	}

	cfg, err := dml.NewConfig("testdata/example.dml")
	if err != nil {
		log.Fatal("❌ Failed to reload config:", err)
	}

	fmt.Println("📦 Full config dump:")
	fmt.Println(cfg.Dump())
}
