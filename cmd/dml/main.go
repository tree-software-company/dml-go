package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tree-software-company/dml-go/dml"
)

type assignments []string

func (a *assignments) String() string {
	return strings.Join(*a, ",")
}

func (a *assignments) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func collectFiles() []string {
	var files []string
	for flag.NArg() > 0 {
		files = append(files, flag.Arg(0))
		flag.CommandLine.Parse(flag.Args()[1:])
	}
	return files
}

func fail(err error) {
	fmt.Printf("Error: %v\n", err)
	os.Exit(1)
}

func main() {
	expandEnv := flag.Bool("env", false, "load .env and expand ${VAR} and ${VAR:-default} references")
	envPrefix := flag.String("env-prefix", "", "override keys from environment variables named PREFIX_KEY (nested keys joined with _)")
	var sets assignments
	flag.Var(&sets, "set", "override a key with key=value, repeatable (for example --set server.port=9000)")
	flag.Parse()

	files := collectFiles()
	if len(files) < 1 {
		fmt.Println("Usage: dml [--env] [--env-prefix PREFIX] [--set key=value]... <file.dml> [more.dml...]")
		os.Exit(1)
	}

	cfg, err := dml.LoadFiles(files...)
	if err != nil {
		fail(err)
	}

	if *expandEnv {
		if err := dml.LoadEnvIfExists(".env"); err != nil {
			fail(err)
		}
		cfg.LoadWithEnv()
	}

	if *envPrefix != "" {
		cfg.EnvOverride(*envPrefix)
	}

	for _, assignment := range sets {
		if err := cfg.SetFromString(assignment); err != nil {
			fail(err)
		}
	}

	fmt.Println("✅ DML file parsed successfully!")
	fmt.Println("\n📄 Config dump:")
	fmt.Println(cfg.Dump())
}
