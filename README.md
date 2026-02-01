# 🧩 DML-Go — Lightweight DML Parser and Config Loader for Go

**DML-Go** is a lightweight and fast Go library that allows you to load, parse, validate, and cache `.dml` (Descriptive Markup Language) configuration files easily.

It supports:

- ✅ Nested structures (`server.port`)
- ✅ Typed access with simple API (`GetString`, `GetInt`, `GetFloat`, `GetBool`, `GetList`, `GetMap`)
- ✅ **Advanced error handling with precise line and column reporting**
- ✅ **Type validation and syntax checking**
- ✅ **Environment variable interpolation and management**
- ✅ **12-factor app support with .env files**
- ✅ Validation of required keys and types
- ✅ In-memory caching for faster reads
- ✅ Manual reload and clear cache functionality
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

| Function                                            | Description                                                        |
| --------------------------------------------------- | ------------------------------------------------------------------ |
| `Load(file string)`                                 | Loads and parses a `.dml` file into a raw `map[string]interface{}` |
| `NewConfig(file string)`                            | Loads and parses a `.dml` file into a `Config` structure           |
| `Cache(file string)`                                | Loads and caches parsed data in memory                             |
| `Reload(file string)`                               | Forces re-parsing and updates the cache for a file                 |
| `ClearCache()`                                      | Clears all cached parsed files from memory                         |
| `Watch(file)`                                       | Live reload of dml file                                            |
| `SetDefaultsToFile(file, variables, isOverwriting)` | Change variables from files to go                                  |

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
| `Dump()`                                         | Dumps the entire parsed data as formatted JSON                   |
| `ValidateRequired(keys...)`                      | Validates that specific keys exist                               |
| `ValidateRequiredTyped(rules map[string]string)` | Validates that keys exist and match expected types               |

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

# Run with coverage
go test ./dml -cover

# Generate coverage report
go test ./dml -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run error handling demo
go run test_errors.go
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
