package dml

import (
    "os"
    "testing"
)

func TestLoadEnv(t *testing.T) {
    envContent := `
# Database configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=admin
DB_PASSWORD="secret123"

# App configuration
APP_NAME='MyApp'
APP_DEBUG=true
`
    
    tmpfile, err := os.CreateTemp("", "test.env")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())
    
    if _, err := tmpfile.Write([]byte(envContent)); err != nil {
        t.Fatal(err)
    }
    tmpfile.Close()
    
    err = LoadEnv(tmpfile.Name())
    if err != nil {
        t.Fatalf("LoadEnv failed: %v", err)
    }
    
    tests := map[string]string{
        "DB_HOST":     "localhost",
        "DB_PORT":     "5432",
        "DB_USER":     "admin",
        "DB_PASSWORD": "secret123",
        "APP_NAME":    "MyApp",
        "APP_DEBUG":   "true",
    }
    
    for key, expected := range tests {
        if got := os.Getenv(key); got != expected {
            t.Errorf("Expected %s=%s, got %s", key, expected, got)
        }
    }
}

func TestLoadEnvIfExists(t *testing.T) {
    err := LoadEnvIfExists("nonexistent.env")
    if err != nil {
        t.Errorf("LoadEnvIfExists should not error on missing file: %v", err)
    }
}

func TestExpandEnv(t *testing.T) {
    os.Setenv("TEST_VAR", "hello")
    os.Setenv("TEST_NUM", "42")
    
    tests := []struct {
        input    string
        expected string
    }{
        {"${TEST_VAR}", "hello"},
        {"$TEST_VAR", "hello"},
        {"Value: ${TEST_VAR}!", "Value: hello!"},
        {"Number: $TEST_NUM", "Number: 42"},
        {"No var", "No var"},
    }
    
    for _, tt := range tests {
        result := ExpandEnv(tt.input)
        if result != tt.expected {
            t.Errorf("ExpandEnv(%q) = %q, want %q", tt.input, result, tt.expected)
        }
    }
}

func TestGetEnv(t *testing.T) {
    os.Setenv("EXISTING_VAR", "value")
    
    if got := GetEnv("EXISTING_VAR", "default"); got != "value" {
        t.Errorf("GetEnv(EXISTING_VAR) = %s, want 'value'", got)
    }
    
    if got := GetEnv("NON_EXISTING", "default"); got != "default" {
        t.Errorf("GetEnv(NON_EXISTING) = %s, want 'default'", got)
    }
}

func TestMustGetEnv(t *testing.T) {
    os.Setenv("REQUIRED_VAR", "present")
    
    value := MustGetEnv("REQUIRED_VAR")
    if value != "present" {
        t.Errorf("MustGetEnv(REQUIRED_VAR) = %s, want 'present'", value)
    }
    
    defer func() {
        if r := recover(); r == nil {
            t.Error("MustGetEnv should panic for missing variable")
        }
    }()
    
    MustGetEnv("MISSING_REQUIRED_VAR")
}

func TestLoadWithEnv(t *testing.T) {
    os.Setenv("TEST_HOST", "example.com")
    os.Setenv("TEST_PORT", "8080")
    
    cfg := New()
    cfg.data = map[string]interface{}{
        "host": "${TEST_HOST}",
        "port": "$TEST_PORT",
        "url":  "https://${TEST_HOST}:${TEST_PORT}",
        "nested": map[string]interface{}{
            "value": "${TEST_HOST}",
        },
        "list": []interface{}{"${TEST_HOST}", "static"},
    }
    
    cfg.LoadWithEnv()
    
    if cfg.GetString("host") != "example.com" {
        t.Errorf("Expected host=example.com, got %s", cfg.GetString("host"))
    }
    
    if cfg.GetString("port") != "8080" {
        t.Errorf("Expected port=8080, got %s", cfg.GetString("port"))
    }
    
    if cfg.GetString("url") != "https://example.com:8080" {
        t.Errorf("Expected url=https://example.com:8080, got %s", cfg.GetString("url"))
    }
}

func TestSetEnvDefaults(t *testing.T) {
    cfg := New()
    cfg.data = map[string]interface{}{
        "app_name":  "MyApp",
        "app_port":  8080,
        "app_debug": true,
        "database": map[string]interface{}{
            "host": "localhost",
            "port": 5432,
        },
    }
    
    err := cfg.SetEnvDefaults("APP")
    if err != nil {
        t.Fatalf("SetEnvDefaults failed: %v", err)
    }
    
    if os.Getenv("APP_APP_NAME") != "MyApp" {
        t.Error("APP_APP_NAME not set correctly")
    }
    
    if os.Getenv("APP_APP_PORT") != "8080" {
        t.Error("APP_APP_PORT not set correctly")
    }
    
    if os.Getenv("APP_DATABASE_HOST") != "localhost" {
        t.Error("APP_DATABASE_HOST not set correctly")
    }
}

func TestEnvOverride(t *testing.T) {
    os.Setenv("APP_NAME", "OverriddenApp")
    os.Setenv("APP_PORT", "9000")
    os.Setenv("APP_DEBUG", "false")
    
    cfg := New()
    cfg.data = map[string]interface{}{
        "name":  "OriginalApp",
        "port":  8080,
        "debug": true,
    }
    
    cfg.EnvOverride("APP")
    
    if cfg.GetString("name") != "OverriddenApp" {
        t.Errorf("Expected name=OverriddenApp, got %s", cfg.GetString("name"))
    }
    
    if cfg.GetInt("port") != 9000 {
        t.Errorf("Expected port=9000, got %d", cfg.GetInt("port"))
    }
    
    if cfg.GetBool("debug") != false {
        t.Errorf("Expected debug=false, got %v", cfg.GetBool("debug"))
    }
}

func TestIntegration_EnvAndDML(t *testing.T) {
    envContent := `
DB_HOST=prod.example.com
DB_PORT=5432
DB_USER=produser
API_KEY=secret-key-123
`
    
    tmpEnv, err := os.CreateTemp("", "integration.env")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpEnv.Name())
    
    tmpEnv.Write([]byte(envContent))
    tmpEnv.Close()

    dmlContent := `
string db_host = "${DB_HOST}";
int db_port = 3306;
string db_user = "${DB_USER}";
string api_key = "${API_KEY}";
string url = "https://${DB_HOST}:${DB_PORT}";
`
    
    tmpDML, err := os.CreateTemp("", "integration.dml")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpDML.Name())
    
    tmpDML.Write([]byte(dmlContent))
    tmpDML.Close()
    
    if err := LoadEnv(tmpEnv.Name()); err != nil {
        t.Fatalf("Failed to load .env: %v", err)
    }
    
    cfg, err := NewConfig(tmpDML.Name())
    if err != nil {
        t.Fatalf("Failed to load DML: %v", err)
    }
    
    cfg.LoadWithEnv()

    if cfg.GetInt("db_port") != 3306 {
        t.Errorf("Expected db_port=3306 before override, got %d", cfg.GetInt("db_port"))
    }
    
    cfg.EnvOverride("")
    
    if cfg.GetInt("db_port") != 5432 {
        t.Errorf("Expected db_port=5432 after override, got %d", cfg.GetInt("db_port"))
    }
    
    if cfg.GetString("db_host") != "prod.example.com" {
        t.Errorf("Expected db_host=prod.example.com, got %s", cfg.GetString("db_host"))
    }
    
    if cfg.GetString("db_user") != "produser" {
        t.Errorf("Expected db_user=produser, got %s", cfg.GetString("db_user"))
    }
    
    if cfg.GetString("url") != "https://prod.example.com:5432" {
        t.Errorf("Expected url with expanded vars, got %s", cfg.GetString("url"))
    }
}