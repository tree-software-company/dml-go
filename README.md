# 🧩 DML-Go — Lightweight DML Parser and Config Loader for Go

**DML-Go** is a lightweight and fast Go library that allows you to load, parse, validate, and cache `.dml` (Descriptive Markup Language) configuration files easily.

It supports:

- ✅ Nested structures (`server.port`)
- ✅ Typed access with simple API (`GetString`, `GetNumber`, `GetBool`, `GetList`, `GetMap`)
- ✅ **Advanced error handling with precise line and column reporting**
- ✅ **Type validation and syntax checking**
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
