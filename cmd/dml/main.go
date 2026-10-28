package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tree-software-company/dml-go/dml"
)

func main() {
	expandEnv := flag.Bool("env", false, "load .env and expand ${VAR} and ${VAR:-default} references")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: dml [--env] <file.dml>")
		os.Exit(1)
	}

	filepath := flag.Arg(0)

	cfg, err := dml.NewConfig(filepath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if *expandEnv {
		if err := dml.LoadEnvIfExists(".env"); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		cfg.LoadWithEnv()
	}

	fmt.Println("✅ DML file parsed successfully!")
	fmt.Println("\n📄 Config dump:")
	fmt.Println(cfg.Dump())
}
