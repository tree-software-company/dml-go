package main

import (
	"fmt"
	"log"

	"github.com/tree-software-company/dml-go/dml"
)

func main() {
	fmt.Println("=== DML Map Style Examples ===\n")

	// Example 1: Global JSON style
	fmt.Println("1. Global JSON Style:")
	dml.SetMapStyle(dml.MapStyleJSON)

	cfg1 := dml.New()
	cfg1.Set("server.port", 8080)
	cfg1.Set("server.timeout", 30)
	cfg1.Set("server.host", "localhost")

	fmt.Println(cfg1.Dump())

	// Example 2: Per-config flat style
	fmt.Println("2. Per-Config Flat Style:")
	cfg2 := dml.New()
	cfg2.SetMapStyle(dml.MapStyleFlat)
	cfg2.Set("database.host", "localhost")
	cfg2.Set("database.port", 5432)

	fmt.Println(cfg2.Dump())

	// Example 3: Auto style (default)
	fmt.Println("3. Auto Style (Smart Decision):")
	dml.SetMapStyle(dml.MapStyleAuto)

	cfg3 := dml.New()
	cfg3.Set("app.name", "MyApp")
	cfg3.Set("app.version", "1.0.0")

	fmt.Println(cfg3.Dump())

	// Example 4: Parse file with @mapStyle directive
	fmt.Println("4. Parse File with @mapStyle Directive:")
	cfg4 := dml.New()
	err := cfg4.Parse(`@mapStyle json

map redis = {
  "host": "localhost",
  "port": 6379
};`)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(cfg4.Dump())

	fmt.Println("✅ All examples completed!")
}
