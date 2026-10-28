package dml

import (
    "os"
    "path/filepath"
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
    
    tests := map[string]string{
        "DB_HOST":     "localhost",
        "DB_PORT":     "5432",
        "DB_USER":     "admin",
        "DB_PASSWORD": "secret123",
        "APP_NAME":    "MyApp",
        "APP_DEBUG":   "true",
    }

    for key := range tests {
        t.Setenv(key, "")
    }
    t.Cleanup(func() {
        for key := range tests {
            delete(fileEnv, key)
        }
    })

    err = LoadEnv(tmpfile.Name())
    if err != nil {
        t.Fatalf("LoadEnv failed: %v", err)
    }

    for key, expected := range tests {
        if got := GetEnv(key, ""); got != expected {
            t.Errorf("Expected %s=%s, got %s", key, expected, got)
        }
        if got := os.Getenv(key); got != "" {
            t.Errorf("Expected %s not to be exported to the process environment", key)
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

func unsetEnvForTest(t *testing.T, key string) {
    t.Helper()
    t.Setenv(key, "")
    os.Unsetenv(key)
    if previous, ok := fileEnv[key]; ok {
        delete(fileEnv, key)
        t.Cleanup(func() { fileEnv[key] = previous })
    }
}

func TestExpandEnvDefaults(t *testing.T) {
    unsetEnvForTest(t, "DML_DEFAULT_UNSET")
    t.Setenv("DML_DEFAULT_SET", "from_env")
    t.Setenv("DML_DEFAULT_EMPTY", "")

    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"unset uses default", "${DML_DEFAULT_UNSET:-default_key}", "default_key"},
        {"set wins over default", "${DML_DEFAULT_SET:-default_key}", "from_env"},
        {"empty uses default", "${DML_DEFAULT_EMPTY:-default_key}", "default_key"},
        {"empty default", "${DML_DEFAULT_UNSET:-}", ""},
        {"url as default", "${DML_DEFAULT_UNSET:-http://localhost:8080}", "http://localhost:8080"},
        {"splits on first operator", "${DML_DEFAULT_UNSET:-a:-b}", "a:-b"},
        {"default with spaces", "${DML_DEFAULT_UNSET:-hello world}", "hello world"},
        {"surrounding text", "pre-${DML_DEFAULT_UNSET:-x}-post", "pre-x-post"},
        {"several in one string", "postgresql://${DML_DEFAULT_UNSET:-admin}@${DML_DEFAULT_SET:-host}/${DML_DEFAULT_EMPTY:-app}", "postgresql://admin@from_env/app"},
        {"braces without default", "${DML_DEFAULT_SET}", "from_env"},
        {"dollar without default", "$DML_DEFAULT_SET", "from_env"},
        {"unset without default", "${DML_DEFAULT_UNSET}", ""},
        {"empty name", "${:-x}", ""},
        {"dollar form ignores operator", "$DML_DEFAULT_UNSET:-x", ":-x"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := ExpandEnv(tt.input); got != tt.expected {
                t.Errorf("ExpandEnv(%q) = %q, want %q", tt.input, got, tt.expected)
            }
        })
    }
}

func TestExpandEnvDefaultsFromDotEnv(t *testing.T) {
    envPath := filepath.Join(t.TempDir(), "defaults.env")
    content := "DML_DEFAULT_DOTENV=from_dotenv\nDML_DEFAULT_DOTENV_EMPTY=\n"
    if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
        t.Fatal(err)
    }

    unsetEnvForTest(t, "DML_DEFAULT_DOTENV")
    unsetEnvForTest(t, "DML_DEFAULT_DOTENV_EMPTY")
    t.Cleanup(func() {
        delete(fileEnv, "DML_DEFAULT_DOTENV")
        delete(fileEnv, "DML_DEFAULT_DOTENV_EMPTY")
    })

    if err := LoadEnv(envPath); err != nil {
        t.Fatalf("LoadEnv failed: %v", err)
    }

    if got := ExpandEnv("${DML_DEFAULT_DOTENV:-fallback}"); got != "from_dotenv" {
        t.Errorf("Expected value from .env file, got %q", got)
    }

    if got := ExpandEnv("${DML_DEFAULT_DOTENV_EMPTY:-fallback}"); got != "fallback" {
        t.Errorf("Expected default for empty .env value, got %q", got)
    }
}

func TestLoadWithEnvDefaults(t *testing.T) {
    unsetEnvForTest(t, "DML_DEFAULT_HOST")
    unsetEnvForTest(t, "DML_DEFAULT_EXTRA")
    unsetEnvForTest(t, "DML_DEFAULT_REGION")

    source := `
string host = "${DML_DEFAULT_HOST:-localhost}";
list hosts = ["a", "${DML_DEFAULT_EXTRA:-example.com}"];
map limits = {"region": "${DML_DEFAULT_REGION:-eu-west-1}"};
`

    load := func() *Config {
        cfg := New()
        if err := cfg.Parse(source); err != nil {
            t.Fatalf("Parse failed: %v", err)
        }
        cfg.LoadWithEnv()
        return cfg
    }

    cfg := load()

    if got := cfg.GetString("host"); got != "localhost" {
        t.Errorf("Expected host=localhost, got %q", got)
    }

    if got := cfg.GetList("hosts"); len(got) != 2 || got[1] != "example.com" {
        t.Errorf("Expected hosts=[a example.com], got %v", got)
    }

    if got := cfg.GetMap("limits")["region"]; got != "eu-west-1" {
        t.Errorf("Expected limits.region=eu-west-1, got %v", got)
    }

    t.Setenv("DML_DEFAULT_HOST", "db.local")
    t.Setenv("DML_DEFAULT_EXTRA", "extra.local")
    t.Setenv("DML_DEFAULT_REGION", "us-east-1")

    cfg = load()

    if got := cfg.GetString("host"); got != "db.local" {
        t.Errorf("Expected host=db.local, got %q", got)
    }

    if got := cfg.GetList("hosts"); len(got) != 2 || got[1] != "extra.local" {
        t.Errorf("Expected hosts=[a extra.local], got %v", got)
    }

    if got := cfg.GetMap("limits")["region"]; got != "us-east-1" {
        t.Errorf("Expected limits.region=us-east-1, got %v", got)
    }
}

func TestLoadWithEnvDefaultsFixture(t *testing.T) {
    for _, key := range []string{
        "API_KEY", "DB_HOST", "DB_USER", "DB_NAME", "OPTIONAL_TOKEN",
        "APP_NAME", "NO_DEFAULT_VAR", "EXTRA_HOST", "REGION",
    } {
        unsetEnvForTest(t, key)
    }

    content, err := os.ReadFile("../testdata/config-with-defaults.dml")
    if err != nil {
        t.Fatal(err)
    }

    load := func() *Config {
        cfg := New()
        if err := cfg.Parse(string(content)); err != nil {
            t.Fatalf("Parse failed: %v", err)
        }
        cfg.LoadWithEnv()
        return cfg
    }

    cfg := load()

    expected := map[string]string{
        "api_key":        "default_key",
        "db_host":        "localhost",
        "db_url":         "postgresql://admin@localhost:5432/app",
        "optional_token": "",
        "app_name":       "MyApp",
        "no_default":     "",
    }
    for key, want := range expected {
        if got := cfg.GetString(key); got != want {
            t.Errorf("%s = %q, want %q", key, got, want)
        }
    }

    t.Setenv("API_KEY", "real_key")
    t.Setenv("DB_HOST", "db.local")

    cfg = load()

    if got := cfg.GetString("api_key"); got != "real_key" {
        t.Errorf("api_key = %q, want real_key", got)
    }

    if got := cfg.GetString("db_url"); got != "postgresql://admin@db.local:5432/app" {
        t.Errorf("db_url = %q, want postgresql://admin@db.local:5432/app", got)
    }
}