# 🧩 DML-Go — Lightweight DML Parser and Config Loader for Go

**DML-Go** is a lightweight and fast Go library that allows you to load, parse, validate, and cache `.dml` (Descriptive Markup Language) configuration files easily.

It supports:

- ✅ Nested structures (`server.port`)
- ✅ Typed access with simple API (`GetString`, `GetNumber`, `GetBool`, `GetList`, `GetMap`)
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

    fmt.Printf("🚀 Starting server on port %d
", port)
    fmt.Printf("⏳ Timeout: %ds
", timeout)

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

## 📚 API Overview

### 🔹 Core functions

| Function                | Description |
|--------------------------|-------------|
| `Load(file string)`      | Loads and parses a `.dml` file into a raw `map[string]interface{}` |
| `NewConfig(file string)` | Loads and parses a `.dml` file into a `Config` structure |
| `Cache(file string)`     | Loads and caches parsed data in memory |
| `Reload(file string)`    | Forces re-parsing and updates the cache for a file |
| `ClearCache()`           | Clears all cached parsed files from memory |
| `Watch(file)`           | Live reload of dml file |

### 🔹 `Config` methods

| Method                      | Description |
|------------------------------|-------------|
| `GetString(key string)`      | Returns a string value (supports nested keys like `server.name`) |
| `GetNumber(key string)`      | Returns a float64 number value |
| `GetBool(key string)`        | Returns a boolean value |
| `GetList(key string)`        | Returns a list or an empty list |
| `GetMap(key string)`         | Returns a map or an empty map |
| `MustString(key string)`     | Returns a string value or panics if missing |
| `Has(key string)`            | Checks if a key exists |
| `Keys()`                     | Returns a sorted list of top-level keys |
| `Dump()`                     | Dumps the entire parsed data as formatted JSON |
| `ValidateRequired(keys...)`  | Validates that specific keys exist |
| `ValidateRequiredTyped(rules map[string]string)` | Validates that keys exist and match expected types |

### 🔹 `Debug` methods

| Method                      | Description |
|------------------------------|-------------|
| `MissedKeys()`      | Schow which variables wasn't declarates in dml config |
| `MissedTypedKeys()`      | Schow which types wasn't declarates in dml config |
| `ValidateState()`      | Schow which types and variables wasn't declarates in dml config |

### 🔹 Internal helpers

| Helper                     | Description |
|-----------------------------|-------------|
| `resolveNestedKey(key)`     | Allows reading deeply nested values using dot notation like `server.port` |

---

## 📚 Example DML Features Supported

```dml
string title = "Hello World";
number age = 25 + 5;
boolean isActive = true;

list hobbies = ["coding", "gaming", "reading"];
map user = { "name": "Szymon", "email": "example@example.com" };

string welcome = "Welcome, " + user.name;
```

- Comments are supported
- Arithmetic operations (`+`, `-`, etc.)
- Nested maps and lists
- String concatenations

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