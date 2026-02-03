package main

import (
	"fmt"
	"log"

	"github.com/tree-software-company/dml-go/dml"
)

func main() {
	fmt.Println("=== DML Map Style Examples ===\n")

	fmt.Println("1. Global JSON Style:")
	dml.SetMapStyle(dml.MapStyleJSON)

	cfg1 := dml.New()
	cfg1.Set("server.port", 8080)
	cfg1.Set("server.timeout", 30)
	cfg1.Set("server.host", "localhost")

	fmt.Println(cfg1.Dump())

	fmt.Println("2. Per-Config Flat Style:")
	cfg2 := dml.New()
	cfg2.SetMapStyle(dml.MapStyleFlat)
	cfg2.Set("database.host", "localhost")
	cfg2.Set("database.port", 5432)

	fmt.Println(cfg2.Dump())

	fmt.Println("3. Auto Style (Smart Decision):")
	dml.SetMapStyle(dml.MapStyleAuto)

	cfg3 := dml.New()
	cfg3.Set("app.name", "MyApp")
	cfg3.Set("app.version", "1.0.0")

	fmt.Println(cfg3.Dump())

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
