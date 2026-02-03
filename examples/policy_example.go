package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tree-software-company/dml-go/dml"
)

func main() {
	fmt.Println("=== DML Default Policy Examples ===\n")

	testFile := "policy_test.dml"
	defer os.Remove(testFile)

	initialConfig := `number port = 8080;
string host = "localhost";`

	err := os.WriteFile(testFile, []byte(initialConfig), 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("1. Permissive Policy (Override All):")
	defaults1 := map[string]any{
		"port":    9000,
		"timeout": 30,
		"debug":   true,
	}

	err = dml.ApplyDefaults(testFile, defaults1, dml.DefaultPolicyPermissive)
	if err != nil {
		log.Fatal(err)
	}

	cfg1, _ := dml.Load(testFile)
	fmt.Printf("Config after permissive: %+v\n\n", cfg1)

	os.WriteFile(testFile, []byte(initialConfig), 0644)

	fmt.Println("2. Strict Policy (Only Missing, Type Check):")
	defaults2 := map[string]any{
		"port":    9000, 
		"timeout": 30,
	}

	err = dml.ApplyDefaults(testFile, defaults2, dml.DefaultPolicyStrict)
	if err != nil {
		log.Fatal(err)
	}

	cfg2, _ := dml.Load(testFile)
	config2 := &dml.Config{}
	config2.FromJSON(mustJSON(cfg2))
	fmt.Printf("Port (unchanged): %d\n", config2.GetInt("port"))
	fmt.Printf("Timeout (added): %d\n\n", config2.GetInt("timeout"))

	fmt.Println("3. Custom Policy:")
	os.WriteFile(testFile, []byte(initialConfig), 0644)

	customPolicy := dml.DefaultPolicy{
		Override:      false,
		StrictTypes:   true,
		OnlyMissing:   true,
		SkipIfPresent: false,
	}

	defaults3 := map[string]any{
		"maxConnections": 100,
		"enableSSL":      true,
	}

	err = dml.ApplyDefaults(testFile, defaults3, customPolicy)
	if err != nil {
		log.Fatal(err)
	}

	cfg3, _ := dml.Load(testFile)
	fmt.Printf("Config with custom policy: %+v\n\n", cfg3)

	fmt.Println("4. Conservative Policy (Skip if any present):")
	os.WriteFile(testFile, []byte(initialConfig), 0644)

	defaults4 := map[string]any{
		"newValue": "test",
	}

	err = dml.ApplyDefaults(testFile, defaults4, dml.DefaultPolicyConservative)
	if err != nil {
		log.Fatal(err)
	}

	cfg4, _ := dml.Load(testFile)
	fmt.Printf("Config (skipped because values present): %+v\n\n", cfg4)

	fmt.Println("✅ All policy examples completed!")
}

func mustJSON(data map[string]any) string {
	cfg := dml.New()
	for k, v := range data {
		cfg.Set(k, v)
	}
	json, _ := cfg.ToJSON()
	return json
}
