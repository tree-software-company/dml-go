package main

import (
    "fmt"
    "os"
    
    "github.com/tree-software-company/dml-go/dml" 
)

func main() {
    fmt.Println("=== DML Error Handling Tests ===\n")
    
    fmt.Println("Test 1: Invalid string format")
    testError(`string name = invalid;`)
    
    fmt.Println("\nTest 2: Invalid integer")
    testError(`int age = abc;`)
    
    fmt.Println("\nTest 3: Invalid identifier")
    testError(`string 123name = "test";`)

    fmt.Println("\nTest 4: Unknown type")
    testError(`unknown value = "test";`)

    fmt.Println("\nTest 5: Invalid boolean")
    testError(`bool active = yes;`)
    
    fmt.Println("\nTest 6: Valid config (should succeed)")
    testValid()
}

func testError(content string) {
    cfg := dml.New()
    err := cfg.Parse(content)
    if err != nil {
        fmt.Println(err)
    } else {
        fmt.Println("❌ Expected error but got none!")
    }
}

func testValid() {
    cfg := dml.New()
    content := `
        string name = "Alice";
        int age = 30;
        float price = 19.99;
        bool active = true;
        list tags = ["go", "dml"];
        map config = {"host": "localhost"};
    `
    
    err := cfg.Parse(content)
    if err != nil {
        fmt.Printf("❌ Unexpected error: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Println("✅ All valid entries parsed successfully!")
    fmt.Printf("  - name: %s\n", cfg.GetString("name"))
    fmt.Printf("  - age: %d\n", cfg.GetInt("age"))
    fmt.Printf("  - price: %.2f\n", cfg.GetFloat("price"))
    fmt.Printf("  - active: %v\n", cfg.GetBool("active"))
}