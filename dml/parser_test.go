package dml

import (
    "testing"
)

func TestParse_ValidString(t *testing.T) {
    cfg := New()
    err := cfg.Parse(`string name = "John Doe";`)
    
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    
    if cfg.GetString("name") != "John Doe" {
        t.Errorf("Expected 'John Doe', got '%s'", cfg.GetString("name"))
    }
}

func TestParse_ValidInt(t *testing.T) {
    cfg := New()
    err := cfg.Parse(`int age = 25;`)
    
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    
    if cfg.GetInt("age") != 25 {
        t.Errorf("Expected 25, got %d", cfg.GetInt("age"))
    }
}

func TestParse_ValidFloat(t *testing.T) {
    cfg := New()
    err := cfg.Parse(`float price = 19.99;`)
    
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    
    if cfg.GetFloat("price") != 19.99 {
        t.Errorf("Expected 19.99, got %f", cfg.GetFloat("price"))
    }
}

func TestParse_ValidBool(t *testing.T) {
    cfg := New()
    err := cfg.Parse(`bool active = true;`)
    
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    
    if !cfg.GetBool("active") {
        t.Error("Expected true, got false")
    }
}

func TestParse_ValidList(t *testing.T) {
    cfg := New()
    err := cfg.Parse(`list tags = ["go", "dml", "parser"];`)
    
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    
    list := cfg.GetList("tags")
    if len(list) != 3 {
        t.Errorf("Expected 3 items, got %d", len(list))
    }
}

func TestParse_ValidMap(t *testing.T) {
    cfg := New()
    err := cfg.Parse(`map config = {"host": "localhost", "port": "8080"};`)
    
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    
    m := cfg.GetMap("config")
    if m["host"] != "localhost" {
        t.Errorf("Expected 'localhost', got '%v'", m["host"])
    }
}

func TestParse_InvalidSyntax(t *testing.T) {
    cfg := New()
    err := cfg.Parse(`string name "John"`)
    
    if err == nil {
        t.Fatal("Expected syntax error")
    }
    
    dmlErr, ok := err.(*DMLError)
    if !ok {
        t.Fatal("Expected DMLError type")
    }
    
    if dmlErr.Type != ErrorTypeSyntax {
        t.Errorf("Expected ErrorTypeSyntax, got %v", dmlErr.Type)
    }
}

func TestParse_InvalidIdentifier(t *testing.T) {
    cfg := New()
    err := cfg.Parse(`string 123invalid = "test";`)
    
    if err == nil {
        t.Fatal("Expected validation error")
    }
    
    dmlErr, ok := err.(*DMLError)
    if !ok {
        t.Fatal("Expected DMLError type")
    }
    
    if dmlErr.Type != ErrorTypeValidation {
        t.Errorf("Expected ErrorTypeValidation, got %v", dmlErr.Type)
    }
}

func TestParse_InvalidType(t *testing.T) {
    cfg := New()
    err := cfg.Parse(`unknown_type value = "test";`)
    
    if err == nil {
        t.Fatal("Expected validation error")
    }
    
    dmlErr, ok := err.(*DMLError)
    if !ok {
        t.Fatal("Expected DMLError type")
    }
    
    if dmlErr.Type != ErrorTypeValidation {
        t.Errorf("Expected ErrorTypeValidation, got %v", dmlErr.Type)
    }
}

func TestParse_InvalidStringFormat(t *testing.T) {
    cfg := New()
    err := cfg.Parse(`string name = invalid;`)
    
    if err == nil {
        t.Fatal("Expected type error")
    }
    
    dmlErr, ok := err.(*DMLError)
    if !ok {
        t.Fatal("Expected DMLError type")
    }
    
    if dmlErr.Type != ErrorTypeType {
        t.Errorf("Expected ErrorTypeType, got %v", dmlErr.Type)
    }
}

func TestParse_InvalidInteger(t *testing.T) {
    cfg := New()
    err := cfg.Parse(`int age = abc;`)
    
    if err == nil {
        t.Fatal("Expected type error")
    }
    
    dmlErr, ok := err.(*DMLError)
    if !ok {
        t.Fatal("Expected DMLError type")
    }
    
    if dmlErr.Type != ErrorTypeType {
        t.Errorf("Expected ErrorTypeType, got %v", dmlErr.Type)
    }
}

func TestParse_InvalidBoolean(t *testing.T) {
    cfg := New()
    err := cfg.Parse(`bool active = yes;`)
    
    if err == nil {
        t.Fatal("Expected type error")
    }
    
    dmlErr, ok := err.(*DMLError)
    if !ok {
        t.Fatal("Expected DMLError type")
    }
    
    if dmlErr.Type != ErrorTypeType {
        t.Errorf("Expected ErrorTypeType, got %v", dmlErr.Type)
    }
}

func TestParse_MultipleLines(t *testing.T) {
    cfg := New()
    content := `
        string name = "Alice";
        int age = 30;
        bool active = true;
    `
    
    err := cfg.Parse(content)
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    
    if cfg.GetString("name") != "Alice" {
        t.Error("Failed to parse name")
    }
    if cfg.GetInt("age") != 30 {
        t.Error("Failed to parse age")
    }
    if !cfg.GetBool("active") {
        t.Error("Failed to parse active")
    }
}

func TestParse_Comments(t *testing.T) {
    cfg := New()
    content := `
        // This is a comment
        string name = "Bob";
        // Another comment
    `
    
    err := cfg.Parse(content)
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    
    if cfg.GetString("name") != "Bob" {
        t.Error("Failed to parse with comments")
    }
}

func TestIsValidIdentifier(t *testing.T) {
    tests := []struct {
        name  string
        valid bool
    }{
        {"validName", true},
        {"_private", true},
        {"name123", true},
        {"CamelCase", true},
        {"123invalid", false},
        {"invalid-name", false},
        {"invalid name", false},
        {"", false},
    }
    
    for _, tt := range tests {
        result := isValidIdentifier(tt.name)
        if result != tt.valid {
            t.Errorf("isValidIdentifier(%q) = %v, want %v", tt.name, result, tt.valid)
        }
    }
}