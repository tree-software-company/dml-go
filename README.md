# 🧩 DML-Go — Lightweight DML Parser and Config Loader for Go

**DML-Go** is a lightweight and fast Go library that allows you to load, parse, validate, and cache `.dml` (Descriptive Markup Language) configuration files easily.

It supports:

- ✅ Nested structures (`server.port`)
- ✅ Typed access with simple API (`GetString`, `GetInt`, `GetFloat`, `GetBool`, `GetList`, `GetMap`)
- ✅ **Advanced error handling with precise line and column reporting**
- ✅ **Type validation and syntax checking**
- ✅ **Environment variable interpolation and management**
- ✅ **12-factor app support with .env files**
- ✅ **Enforced map style (JSON/Flat/Auto) for consistent output**
- ✅ **Smart default policies for configuration management**
- ✅ Validation of required keys and types
- ✅ In-memory caching for faster reads
- ✅ Manual reload and clear cache functionality
- ✅ **Partial reload with `ReloadKeys` — hot-reload only selected keys (perfect for long-running services)**
- ✅ Full nested key support (e.g., `server.port`)

Built for configuration-driven applications and servers.

---

## 📦 Installation

Clone the repository:

```bash
git clone https://github.com/tree-software-company/dml-go.git
cd dml-go
```

Or copy the `dml/` folder into your Go project.

---

## 🚀 Quick Start

### 1. Example `config.dml`

```dml
map server = {
  "port": 8080,
  "timeout": 15
};

map database = {
  "host": "localhost",
  "port": 5432
};
```

Save as `testdata/config.dml`.

---

### 2. Basic usage

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/tree-software-company/dml-go/dml"
)

func main() {
    cfg, err := dml.NewConfig("testdata/config.dml")
    if err != nil {
        log.Fatal(err)
    }

    serverMap := cfg.GetMap("server")
    port := int(serverMap["port"].(float64))
    timeout := int(serverMap["timeout"].(float64))

    fmt.Printf("🚀 Starting server on port %d\n", port)
    fmt.Printf("⏳ Timeout: %ds\n", timeout)

    server := &http.Server{
        Addr:         fmt.Sprintf(":%d", port),
        ReadTimeout:  time.Duration(timeout) * time.Second,
        WriteTimeout: time.Duration(timeout) * time.Second,
    }

    http.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "👋 Hello from DML-configured server!")
    })

    if err := server.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
```

Then open:  
`http://localhost:8080/api/hello`

✅ You will see: `"👋 Hello from DML-configured server!"`

---

## 🎯 Default Policies

DML-Go provides a powerful and flexible system for applying default values to configurations. Instead of scattered boolean flags, use a single `DefaultPolicy` struct.

### The Problem

Before:

```go
// ❌ Hard to understand, easy to make mistakes
ApplyDefaults(file, defaults, true, false, true, false)
// What does each boolean mean? 🤔
```

After:

```go
// ✅ Crystal clear intentions
dml.ApplyDefaults(file, defaults, dml.DefaultPolicy{
    OnlyMissing: true,
    StrictTypes: true,
})
```

### DefaultPolicy Structure

```go
type DefaultPolicy struct {
    Override      bool // Override existing values
    StrictTypes   bool // Enforce type matching
    OnlyMissing   bool // Only set values that don't exist
    SkipIfPresent bool // Skip if any value is already present
}
```

### Predefined Policies

DML-Go includes three ready-to-use policies:

#### 1. **Permissive Policy** - Override Everything

```go
dml.ApplyDefaults("config.dml", defaults, dml.DefaultPolicyPermissive)
```

**Behavior:**

- ✅ Overrides all existing values
- ✅ No type checking
- ✅ Always applies defaults
- ⚠️ Use with caution in production

**Use case:** Development, testing, resetting configuration

#### 2. **Strict Policy** - Safe Defaults Only

```go
dml.ApplyDefaults("config.dml", defaults, dml.DefaultPolicyStrict)
```

**Behavior:**

- ✅ Only adds missing values
- ✅ Enforces type matching
- ✅ Never overrides existing values
- ✅ Production-safe

**Use case:** Production deployments, safe migrations

#### 3. **Conservative Policy** - Skip if Present

```go
dml.ApplyDefaults("config.dml", defaults, dml.DefaultPolicyConservative)
```

**Behavior:**

- ✅ Skips if ANY value exists
- ✅ Type checking enabled
- ✅ Preserves existing configurations
- ✅ Ultra-safe

**Use case:** First-time initialization only

### Custom Policies

Create your own policy for specific needs:

```go
customPolicy := dml.DefaultPolicy{
    Override:      false,  // Don't overwrite
    StrictTypes:   true,   // Match types
    OnlyMissing:   true,   // Only add missing
    SkipIfPresent: false,  // Don't skip on existing
}

dml.ApplyDefaults("config.dml", defaults, customPolicy)
```

### Real-World Examples

#### Example 1: Initialize New Configuration

```go
defaults := map[string]any{
    "port":           8080,
    "timeout":        30,
    "debug":          false,
    "maxConnections": 100,
}

// Use permissive for first-time setup
err := dml.ApplyDefaults("config.dml", defaults, dml.DefaultPolicyPermissive)
```

#### Example 2: Add Missing Settings to Production

```go
newDefaults := map[string]any{
    "cacheEnabled":  true,
    "cacheTTL":      3600,
    "rateLimitRPS":  100,
}

// Use strict to avoid breaking existing config
err := dml.ApplyDefaults("production.dml", newDefaults, dml.DefaultPolicyStrict)
```

#### Example 3: Conditional Initialization

```go
defaults := map[string]any{
    "firstRun": true,
    "version":  "1.0.0",
}

// Only apply if config is completely empty
err := dml.ApplyDefaults("config.dml", defaults, dml.DefaultPolicyConservative)
```

#### Example 4: Type-Safe Migration

```go
// Current config has: port = 8080 (int)
// This will FAIL because of type mismatch
defaults := map[string]any{
    "port": "9000", // ❌ string instead of int
}

err := dml.ApplyDefaults("config.dml", defaults, dml.DefaultPolicy{
    StrictTypes: true,
    Override:    true,
})
// Error: type mismatch for key 'port': expected int, got string
```

### Policy Comparison Table

| Policy           | Override | StrictTypes | OnlyMissing | SkipIfPresent | Best For                  |
| ---------------- | -------- | ----------- | ----------- | ------------- | ------------------------- |
| **Permissive**   | ✅       | ❌          | ❌          | ❌            | Development, testing      |
| **Strict**       | ❌       | ✅          | ✅          | ❌            | Production, safe updates  |
| **Conservative** | ❌       | ✅          | ❌          | ✅            | First-time initialization |
| **Custom**       | 🎛️       | 🎛️          | 🎛️          | 🎛️            | Specific requirements     |

### Testing Policies

```go
package main

import (
    "testing"
    "github.com/tree-software-company/dml-go/dml"
)

func TestDefaultPolicies(t *testing.T) {
    // Test permissive
    defaults := map[string]any{"port": 9000}
    err := dml.ApplyDefaults("test.dml", defaults, dml.DefaultPolicyPermissive)
    // Should override existing port value

    // Test strict
    err = dml.ApplyDefaults("test.dml", defaults, dml.DefaultPolicyStrict)
    // Should keep existing port value

    // Test conservative
    err = dml.ApplyDefaults("test.dml", defaults, dml.DefaultPolicyConservative)
    // Should skip entirely if any value exists
}
```

### Error Handling

```go
defaults := map[string]any{
    "port": "invalid", // Wrong type
}

err := dml.ApplyDefaults("config.dml", defaults, dml.DefaultPolicyStrict)
if err != nil {
    // Error: type mismatch for key 'port': expected int, got string
    log.Printf("Policy violation: %v", err)
}
```

---

## 🎨 Map Style Control

DML-Go gives you full control over how configuration is dumped - no more surprises!

### Global Map Style

Set the style for all dumps:

```go
// Always use JSON-style maps
dml.SetMapStyle(dml.MapStyleJSON)

cfg := dml.New()
cfg.Set("server.port", 8080)
cfg.Set("server.host", "localhost")

fmt.Println(cfg.Dump())
```

**Output:**

```dml
@mapStyle json

map server = {
  "host": "localhost",
  "port": 8080
};
```

### Available Styles

| Style          | Behavior                               | Example                            |
| -------------- | -------------------------------------- | ---------------------------------- |
| `MapStyleJSON` | Always uses map syntax                 | `map server = { "port": 8080 };`   |
| `MapStyleFlat` | Always uses flat key-value syntax      | `number server.port = 8080;`       |
| `MapStyleAuto` | Automatically decides based on content | Smart decision based on complexity |

### Per-Config Override

Override style for specific config instances:

```go
cfg := dml.New()
cfg.SetMapStyle(dml.MapStyleFlat)
cfg.Set("database.host", "localhost")
cfg.Set("database.port", 5432)

fmt.Println(cfg.Dump())
```

**Output:**

```dml
@mapStyle flat

string database.host = "localhost";
number database.port = 5432;
```

### DML File Directive

Control style directly in `.dml` files:

```dml
@mapStyle json

map server = {
  "port": 8080,
  "timeout": 30
};
```

The parser respects the `@mapStyle` directive and maintains consistency.

### Why Map Style Control?

**Problem:** CLI tools might generate inconsistent output:

```dml
// Sometimes this:
string server.port = "8080";

// Sometimes this:
map server = {
  "port": 8080
};
```

**Solution:** Enforce consistent style:

```go
dml.SetMapStyle(dml.MapStyleJSON)
// Now ALWAYS generates map syntax - zero surprises! 🎯
```

---

## 🌍 Environment Variables Integration

DML-Go provides powerful environment variable support for 12-factor apps.

### Loading .env Files

```go
package main

import (
    "fmt"
    "log"

    "github.com/tree-software-company/dml-go/dml"
)

func main() {
    // Load .env file
    if err := dml.LoadEnv(".env"); err != nil {
        log.Fatal(err)
    }

    // Or load only if exists (no error if missing)
    if err := dml.LoadEnvIfExists(".env"); err != nil {
        log.Fatal(err)
    }

    fmt.Println("✅ Environment variables loaded!")
}
```

### Example `.env` file

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=admin
DB_PASSWORD=secret123
DB_NAME=myapp
APP_NAME=DML-Go App
APP_ENV=production
API_KEY=your-api-key-here
```

---

### Environment Variable Interpolation

Use `${VAR_NAME}` syntax in your DML files:

```dml
// config.dml with environment variables
string db_host = "${DB_HOST}";
int db_port = 5432;
string db_user = "${DB_USER}";
string db_password = "${DB_PASSWORD}";
string db_name = "${DB_NAME}";

// Interpolation works in complex strings too
string db_url = "postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}";

string app_name = "${APP_NAME}";
string api_key = "${API_KEY}";

// Lists with env vars
list allowed_hosts = ["localhost", "${DB_HOST}", "example.com"];
```

**Load and expand variables:**

```go
// Load environment variables first
dml.LoadEnv(".env")

// Load DML config
cfg, err := dml.NewConfig("config.dml")
if err != nil {
    log.Fatal(err)
}

// Expand environment variables in config
cfg.LoadWithEnv()

// Access expanded values
fmt.Println(cfg.GetString("db_url"))
// Output: postgresql://admin:secret123@localhost:5432/myapp
```

---

### Environment Variable Helpers

```go
// Get environment variable with default value
timeout := dml.GetEnv("REQUEST_TIMEOUT", "30")

// Get environment variable or panic if not set
apiKey := dml.MustGetEnv("API_KEY")

// Expand environment variables in any string
url := dml.ExpandEnv("https://${HOST}:${PORT}/api")
```

---

### Setting Environment Defaults from Config

```go
cfg, _ := dml.NewConfig("config.dml")

// Set environment variables from config with prefix
cfg.SetEnvDefaults("SERVER")
// Sets: SERVER_PORT, SERVER_HOST, SERVER_TIMEOUT, etc.
```

---

### Override Config from Environment

```go
cfg, _ := dml.NewConfig("config.dml")

// Override config values from environment variables
// Will look for env vars matching config keys (uppercase)
cfg.EnvOverride("")

// With prefix: APP_PORT, APP_DEBUG, etc.
cfg.EnvOverride("APP")
```

**Example:**

```go
// config.dml
int port = 8080;
bool debug = true;

// Set environment variables
os.Setenv("PORT", "9000")
os.Setenv("DEBUG", "false")

// Override from environment
cfg.EnvOverride("")

// Values are now from environment
fmt.Println(cfg.GetInt("port"))   // 9000
fmt.Println(cfg.GetBool("debug")) // false
```

---

### Full 12-Factor App Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/tree-software-company/dml-go/dml"
)

func main() {
    // 1. Load .env file (development)
    if err := dml.LoadEnvIfExists(".env"); err != nil {
        log.Fatal(err)
    }

    // 2. Load DML config
    cfg, err := dml.NewConfig("config.dml")
    if err != nil {
        log.Fatal(err)
    }

    // 3. Expand environment variables
    cfg.LoadWithEnv()

    // 4. Override from environment (production overrides)
    cfg.EnvOverride("")

    // 5. Use configuration
    fmt.Printf("🚀 Starting %s\n", cfg.GetString("app_name"))
    fmt.Printf("🌍 Environment: %s\n", dml.GetEnv("APP_ENV", "development"))
    fmt.Printf("💾 Database: %s\n", cfg.GetString("db_host"))
    fmt.Printf("🔌 Port: %d\n", cfg.GetInt("port"))
}
```

---

## ♻️ Partial Reload — `ReloadKeys`

For **long-running Go services** you often want to hot-reload a small subset of
configuration (e.g. rate limits, feature flags) without touching the rest.
`ReloadKeys` re-parses the file but updates **only the keys you list**.

### Signatures

```go
// Package-level — updates the global in-memory cache.
func ReloadKeys(filepath string, keys ...string) (map[string]any, error)

// Method — updates a *Config instance directly.
func (c *Config) ReloadKeys(filepath string, keys ...string) error
```

### Behaviour

| Scenario                                   | Result                                        |
| ------------------------------------------ | --------------------------------------------- |
| Key exists in file and is listed           | Value updated in cache / `*Config`            |
| Key exists in file but **not** listed      | Untouched — existing value preserved          |
| Key listed but **absent** from file        | Silently skipped — existing value preserved   |
| No cache entry yet (package-level variant) | New entry created containing only listed keys |

Thread-safe: both variants use `cacheMutex` for the cache write.

### Example — `*Config` method (recommended for services)

```go
package main

import (
    "log"
    "time"

    "github.com/tree-software-company/dml-go/dml"
)

func main() {
    cfg, err := dml.NewConfig("config.dml")
    if err != nil {
        log.Fatal(err)
    }

    // Poll every 30 s and hot-reload only server + database.
    // app_version (and any other key) is never touched.
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        if err := cfg.ReloadKeys("config.dml", "server", "database"); err != nil {
            log.Printf("reload error: %v", err)
        }
    }
}
```

### Example — package-level (cache-based)

```go
// Somewhere at startup:
dml.Cache("config.dml")

// Later, on a timer or signal:
data, err := dml.ReloadKeys("config.dml", "server", "database")
if err != nil {
    log.Printf("reload error: %v", err)
}
srv := data["server"].(map[string]any)
log.Printf("server.host = %v", srv["host"])
```

### When to use `Reload` vs `ReloadKeys`

| Situation                                            | Use                   |
| ---------------------------------------------------- | --------------------- |
| Small config, full refresh is fine                   | `Reload`              |
| Large config, only a few keys change at runtime      | `ReloadKeys`          |
| Config has computed/enriched keys you do not persist | `ReloadKeys`          |
| First load at startup                                | `NewConfig` / `Cache` |

---

## 🔍 Error Handling & Validation

DML-Go provides comprehensive error handling with detailed context about syntax and validation errors.

### Error Types

| Error Type           | Description                        | Example                            |
| -------------------- | ---------------------------------- | ---------------------------------- |
| **Syntax Error**     | Invalid DML syntax                 | Missing `=` or semicolon           |
| **Validation Error** | Invalid identifier or unknown type | Variable name starting with number |
| **Type Error**       | Value doesn't match declared type  | String without quotes              |

### Example Error Output

```go
cfg := dml.New()
err := cfg.Parse(`string name = invalid;`)
if err != nil {
    fmt.Println(err)
}
```

**Output:**

```
Type Error at line 1:18
  String must be enclosed in double quotes

  string name = invalid;
                 ^
```

### Common Validation Rules

#### ✅ Valid Variable Names

```dml
string userName = "Alice";      // ✅ Valid
string _private = "secret";     // ✅ Valid
string data123 = "info";        // ✅ Valid
```

#### ❌ Invalid Variable Names

```dml
string 123name = "test";        // ❌ Cannot start with number
string user-name = "test";      // ❌ No hyphens allowed
string user name = "test";      // ❌ No spaces allowed
```

#### ✅ Valid Type Declarations

```dml
string name = "John";           // ✅ Strings must have quotes
int age = 25;                   // ✅ Integer number
float price = 19.99;            // ✅ Floating point number
bool active = true;             // ✅ Boolean: true or false
list tags = ["go", "dml"];      // ✅ List of values
map config = {"key": "value"};  // ✅ Key-value map
```

#### ❌ Common Type Errors

```dml
string name = John;             // ❌ Missing quotes
int age = abc;                  // ❌ Not a valid integer
bool active = yes;              // ❌ Must be 'true' or 'false'
unknown_type value = "test";    // ❌ Unknown type
```

### Testing Error Handling

```go
package main

import (
    "fmt"
    "github.com/tree-software-company/dml-go/dml"
)

func main() {
    cfg := dml.New()

    // This will return a detailed error
    err := cfg.Parse(`int age = invalid_number;`)

    if err != nil {
        // Check if it's a DMLError for detailed info
        if dmlErr, ok := err.(*dml.DMLError); ok {
            fmt.Printf("Error Type: %s\n", dmlErr.Type)
            fmt.Printf("Line: %d, Column: %d\n", dmlErr.Line, dmlErr.Column)
            fmt.Printf("Message: %s\n", dmlErr.Message)
        }

        // Or just print the full formatted error
        fmt.Println(err)
    }
}
```

---

## 📚 API Overview

### 🔹 Core functions

| Function                                  | Description                                                        |
| ----------------------------------------- | ------------------------------------------------------------------ |
| `Load(file string)`                       | Loads and parses a `.dml` file into a raw `map[string]interface{}` |
| `NewConfig(file string)`                  | Loads and parses a `.dml` file into a `Config` structure           |
| `Cache(file string)`                      | Loads and caches parsed data in memory                             |
| `Reload(file string)`                     | Forces re-parsing and updates the cache for a file                 |
| `ReloadKeys(file string, keys ...string)` | Partially reloads only the given top-level keys in the cache       |
| `ClearCache()`                            | Clears all cached parsed files from memory                         |
| `Watch(file)`                             | Live reload of dml file                                            |
| `ApplyDefaults(file, defaults, policy)`   | Apply default values with policy control                           |
| `SetMapStyle(style MapStyle)`             | Sets global map dump style (JSON/Flat/Auto)                        |
| `GetMapStyle()`                           | Returns current global map style                                   |

### 🔹 `Config` methods

| Method                                           | Description                                                      |
| ------------------------------------------------ | ---------------------------------------------------------------- |
| `Parse(content string)`                          | Parses DML content string with validation                        |
| `GetString(key string)`                          | Returns a string value (supports nested keys like `server.name`) |
| `GetInt(key string)`                             | Returns an integer value                                         |
| `GetFloat(key string)`                           | Returns a float64 number value                                   |
| `GetBool(key string)`                            | Returns a boolean value                                          |
| `GetList(key string)`                            | Returns a list or an empty list                                  |
| `GetMap(key string)`                             | Returns a map or an empty map                                    |
| `MustString(key string)`                         | Returns a string value or panics if missing                      |
| `Has(key string)`                                | Checks if a key exists                                           |
| `Keys()`                                         | Returns a sorted list of top-level keys                          |
| `Dump()`                                         | Dumps the entire parsed data in DML format (respects map style)  |
| `SetMapStyle(style MapStyle)`                    | Sets map style for this specific config                          |
| `ReloadKeys(file string, keys ...string)`        | Hot-reloads only the specified top-level keys from a file        |
| `ValidateRequired(keys...)`                      | Validates that specific keys exist                               |
| `ValidateRequiredTyped(rules map[string]string)` | Validates that keys exist and match expected types               |

### 🔹 Default Policy Presets

| Policy                      | Description                                    |
| --------------------------- | ---------------------------------------------- |
| `DefaultPolicyPermissive`   | Override all, no type checking (dev/testing)   |
| `DefaultPolicyStrict`       | Only missing, strict types (production-safe)   |
| `DefaultPolicyConservative` | Skip if any present, strict types (ultra-safe) |

### 🔹 `Debug` methods

| Method              | Description                                                   |
| ------------------- | ------------------------------------------------------------- |
| `MissedKeys()`      | Show which variables weren't declared in dml config           |
| `MissedTypedKeys()` | Show which types weren't declared in dml config               |
| `ValidateState()`   | Show which types and variables weren't declared in dml config |

### 🔹 Error types

| Type                  | Description                                       |
| --------------------- | ------------------------------------------------- |
| `DMLError`            | Structured error with line, column, and context   |
| `ErrorTypeSyntax`     | Syntax errors (missing operators, brackets, etc.) |
| `ErrorTypeValidation` | Validation errors (invalid identifiers, etc.)     |
| `ErrorTypeType`       | Type mismatch errors (wrong value format)         |

### 🔹 Internal helpers

| Helper                    | Description                                                               |
| ------------------------- | ------------------------------------------------------------------------- |
| `resolveNestedKey(key)`   | Allows reading deeply nested values using dot notation like `server.port` |
| `isValidIdentifier(name)` | Validates variable names according to DML rules                           |

---

## 📚 Example DML Features Supported

```dml
// Control dump style with directive
@mapStyle json

// Strings must be in double quotes
string title = "Hello World";

// Numbers can be integers or floats
int age = 25;
float price = 19.99;

// Booleans must be true or false
bool isActive = true;

// Lists use square brackets
list hobbies = ["coding", "gaming", "reading"];

// Maps use curly braces with key-value pairs
map user = {"name": "Szymon", "email": "example@example.com"};

// Comments are supported (use //)
// This is a comment
```

### Supported Types

| Type     | Format             | Example            |
| -------- | ------------------ | ------------------ |
| `string` | Double-quoted text | `"Hello World"`    |
| `int`    | Integer number     | `42`               |
| `float`  | Decimal number     | `3.14`             |
| `bool`   | true or false      | `true`             |
| `list`   | Square brackets    | `["a", "b", "c"]`  |
| `map`    | Curly braces       | `{"key": "value"}` |

---

## 🧪 Testing

Run the test suite:

```bash
# Run all tests
go test ./dml -v

# Run specific tests
go test ./dml -run TestDefaultPolicy -v

# Run with coverage
go test ./dml -cover

# Generate coverage report
go test ./dml -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run examples
go run examples/policy_example.go
go run examples/mapstyle_example.go
go run examples/env_example.go
go run examples/reload_keys_example.go

# Run error handling demo
go run tests/test_errors.go

# Test map style functionality
go run tests/test_mapstyle.go
```

## 🧰 Lint — static DML checks

A simple file-based linter was added to catch common DML mistakes early.

- Function: `func Lint(file string) ([]LintIssue, error)`
- Type:
  ```go
  type LintIssue struct {
    Level   string // "error" | "warning"
    Code    string // MAP_TRAILING_COMMA, MIXED_MAP_STYLE, TYPED_MAP_ENTRY, UNUSED_DEFAULT, EMPTY_MAP, MAP_UNCLOSED
    Message string
    Line    int
  }
  ```

Checks performed:

- ❌ MAP_TRAILING_COMMA — trailing comma after last map element
- ❌ TYPED_MAP_ENTRY — typed entries inside maps (e.g. `string port = ...`)
- ⚠️ MIXED_MAP_STYLE — mixed style: maps and root-level vars used together
- ⚠️ UNUSED_DEFAULT — default declarations that are never used
- ⚠️ EMPTY_MAP — empty maps (e.g. `server = {}`)
- ❌ MAP_UNCLOSED — unclosed map (missing `}`)

Usage example:

```go
issues, err := dml.Lint("testdata/config.dml")
if err != nil {
  log.Fatal(err)
}
for _, it := range issues {
  fmt.Printf("%s: %s (code=%s) line=%d\n", it.Level, it.Message, it.Code, it.Line)
}
```

Files:

- Implementation: `dml/lint.go`
- Tests: `dml/lint_test.go`

Run linter tests:

```bash
cd /Users/szymonmastalerz/Documents/_prywatne-studia/tree/dml-go
go test ./dml -run TestLint_BasicChecks -v
# or run all tests
go test ./... -v
```

Consider adding a CLI command or CI job to run `Lint` on DML files to prevent common config bugs before deployment.

### Test Coverage

The library includes comprehensive tests for:

- ✅ Valid syntax parsing for all types
- ✅ Invalid syntax detection
- ✅ Type validation
- ✅ Identifier validation
- ✅ Multi-line configurations
- ✅ Comment handling
- ✅ Error message formatting
- ✅ Map style enforcement (JSON/Flat/Auto)
- ✅ Default policy behavior (Permissive/Strict/Conservative)
- ✅ Type-safe default application
- ✅ Policy violation detection

---

## 📄 License

This project is licensed under the [Apache 2.0 License](LICENSE).

---

## 👤 Author

Developed by [Tree Software Company](https://github.com/tree-software-company) ✨

---

## 📣 Contributions

Feel free to open issues, create pull requests, or suggest features! 🚀
Let's make DML integration in Go even better together!

---

## 🐛 Bug Reports

When reporting bugs, please include:

1. Your DML configuration file
2. The error message with line and column numbers
3. Expected vs actual behavior
4. Go version and OS

This helps us fix issues faster! 💪
