package main

import (
	"fmt"
	"os"

	"github.com/tree-software-company/dml-go/dml"
)

func main() {
	// Test 1: Global JSON style
	fmt.Println("=== Test 1: Global JSON Style ===")
	dml.SetMapStyle(dml.MapStyleJSON)

	cfg1 := dml.New()
	cfg1.Set("server.port", 8080)
	cfg1.Set("server.host", "localhost")
	cfg1.Set("server.timeout", 30)

	output1 := cfg1.Dump()
	fmt.Println(output1)
	os.WriteFile("output_json.dml", []byte(output1), 0644)

	// Test 2: Flat style
	fmt.Println("=== Test 2: Flat Style ===")
	dml.SetMapStyle(dml.MapStyleFlat)

	cfg2 := dml.New()
	cfg2.Set("database.host", "localhost")
	cfg2.Set("database.port", 5432)
	cfg2.Set("database.name", "mydb")

	output2 := cfg2.Dump()
	fmt.Println(output2)
	os.WriteFile("output_flat.dml", []byte(output2), 0644)

	// Test 3: Auto style (default behavior)
	fmt.Println("=== Test 3: Auto Style ===")
	dml.SetMapStyle(dml.MapStyleAuto)

	cfg3 := dml.New()
	cfg3.Set("app.name", "MyApp")
	cfg3.Set("app.version", "1.0.0")

	output3 := cfg3.Dump()
	fmt.Println(output3)
	os.WriteFile("output_auto.dml", []byte(output3), 0644)

	// Test 4: Per-config style override
	fmt.Println("=== Test 4: Per-Config Override ===")
	cfg4 := dml.New()
	cfg4.SetMapStyle(dml.MapStyleJSON)
	cfg4.Set("redis.host", "localhost")
	cfg4.Set("redis.port", 6379)

	output4 := cfg4.Dump()
	fmt.Println(output4)
	os.WriteFile("output_override.dml", []byte(output4), 0644)

	fmt.Println("\n✅ All tests completed. Check the generated .dml files.")
}
