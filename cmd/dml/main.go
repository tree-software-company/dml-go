package main

import (
	"fmt"
	"os"

	"github.com/tree-software-company/dml-go/dml"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: dml <file.dml>")
		os.Exit(1)
	}

	filepath := os.Args[1]

	cfg, err := dml.NewConfig(filepath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ DML file parsed successfully!")
	fmt.Println("\n📄 Config dump:")
	fmt.Println(cfg.Dump())
}
