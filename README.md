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

**Default values:** `${VAR:-default}` expands to `default` when `VAR` is unset or empty (values from `.env` count too):

```dml
string api_key = "${API_KEY:-default_key}";
string db_url = "postgresql://${DB_USER:-admin}@${DB_HOST:-localhost}:5432/${DB_NAME:-app}";
string optional_token = "${OPTIONAL_TOKEN:-}";
list allowed_hosts = ["localhost", "${EXTRA_HOST:-example.com}"];
map limits = {"region": "${REGION:-eu-west-1}"};
```

Limitations:

- The default ends at the first `}`, so nesting such as `${A:-${B}}` is not supported.
- The default is inserted literally; `${A:-$B}` yields the text `$B`.
- Expansion runs after parsing, so defaults work in `string` values, list items and map values, not in `int`, `float` or `bool` declarations.
- Do not put a comma in a default inside a `list`.
- Only the `:-` form exists; `-`, `:=` and `:?` are not supported. The `$VAR` form has no default syntax.

**CLI:** `dml file.dml` prints the file unchanged. Add `--env` to load `.env` from the current directory and expand variables before dumping:

```bash
go run ./cmd/dml --env testdata/config-with-defaults.dml
API_KEY=real_key go run ./cmd/dml --env testdata/config-with-defaults.dml
```

The dump contains the expanded values, so avoid running it on files that reference real secrets.

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

## 🧱 Layered Configuration — `LoadFiles`, `Merge`, `SetFromString`

Keep one base file and override it per environment (`config.dml`, then `config.prod.dml`), then let environment variables and CLI flags have the last word.

Precedence, lowest to highest:

1. Files, in the order given (a later file overrides an earlier one)
2. Environment variables (`EnvOverride`, `--env-prefix`)
3. Explicit assignments (`SetFromString`, `--set`)

### Library

```go
cfg, err := dml.LoadFiles("config.dml", "config.prod.dml")
if err != nil {
    log.Fatal(err)
}

cfg.EnvOverride("APP")

if err := cfg.SetFromString("server.port=7000"); err != nil {
    log.Fatal(err)
}
```

`Config.Merge(other)` does the same file-to-file step on two `*Config` values you already have.

### Merge rules

- Maps are merged key by key, recursively, so sibling keys from the base file survive.
- Scalars and lists from the overriding file replace the base value; lists are not appended.
- If the overriding file uses a different type for a key, the overriding file wins.
- The result never shares maps or lists with the merged-in config.
- An explicit `@mapStyle` in the overriding file replaces the base map style.

### `SetFromString`

`key=value` is split on the first `=`; the key may be dotted (`server.port`).

- If the key exists, the value is converted to its type (`int`, `float`, `bool` or `string`); a value that does not fit returns `dml.ErrInvalidOverride` without echoing the value.
- A key that holds a map or list cannot be replaced from a string.
- A new key gets an inferred type: `int`, then `float`, then `true`/`false`, otherwise `string`.
- A malformed assignment returns `dml.ErrInvalidAssignment`.

### Safety

Each file passes the same path checks as `NewConfig` (`dml.ErrInvalidPath`) and is limited to 10 MiB (`dml.ErrLayerTooLarge`). Parse errors keep `Type`, `Line` and `Column` but carry a fixed message and no source line, so values from override files such as `config.prod.dml` never reach your logs. `LoadFiles()` without paths returns `dml.ErrNoLayers`.

### CLI

```bash
go run ./cmd/dml testdata/config.base.dml testdata/config.prod.dml
APP_SERVER_PORT=9000 go run ./cmd/dml --env-prefix APP testdata/config.base.dml testdata/config.prod.dml
go run ./cmd/dml --set server.port=7000 --set debug=true testdata/config.base.dml testdata/config.prod.dml
```

`--set` can be repeated, and flags may appear before or after the file names. `--env-prefix` needs a non-empty prefix: `APP` maps `server.port` to `APP_SERVER_PORT`. Only keys already present in the loaded files are overridden by environment variables; use `--set` to add new keys.

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

## 👀 Live Reload — `Watch`

`Watch` is **opt-in**. It polls one `.dml` file and, when it changes, parses it
into a brand-new `Config` and swaps that in under a `sync.RWMutex`. Your service
keeps running and never needs a restart.

### Signatures

```go
type WatchOptions struct {
    Interval    time.Duration
    MaxFileSize int64
    OnReload    func(*Config)
    OnError     func(error)
}

func Watch(filepath string, opts WatchOptions) (*Watcher, error)

func (w *Watcher) Config() *Config
func (w *Watcher) Get(key string) (any, bool)
func (w *Watcher) GetString(key string) string
func (w *Watcher) GetInt(key string) int
func (w *Watcher) GetFloat(key string) float64
func (w *Watcher) GetBool(key string) bool
func (w *Watcher) GetList(key string) []any
func (w *Watcher) GetMap(key string) map[string]any
func (w *Watcher) Has(key string) bool
func (w *Watcher) Keys() []string
func (w *Watcher) Stop()
```

### Options

| Field         | Default | Meaning                                                      |
| ------------- | ------- | ------------------------------------------------------------ |
| `Interval`    | `1s`    | How often the file is checked                                |
| `MaxFileSize` | `10 MiB` | Larger files are rejected with `ErrWatchFileTooLarge`       |
| `OnReload`    | none    | Called with the new `*Config` after every successful reload  |
| `OnError`     | none    | Called when a reload fails; the previous config stays active |

### Behaviour

| Scenario                                 | Result                                                              |
| ---------------------------------------- | ------------------------------------------------------------------- |
| File edited in place                     | New `Config` is installed once the file is stable for one interval  |
| File replaced by rename (atomic save)    | Detected, because the file identity changes                         |
| New content has a syntax / type error    | `OnError` is called once, last good config stays active             |
| File deleted                             | `OnError` is called once, last good config stays active             |
| File reappears or is fixed               | Reloaded and `OnReload` is called                                   |
| File larger than `MaxFileSize`           | `OnError` receives `ErrWatchFileTooLarge`, last good config stays   |
| Invalid path (`..`, NUL, empty)          | `Watch` returns an error wrapping `ErrInvalidPath`                  |

### Thread safety

- The `Watcher` getters are safe to call from any goroutine while reloads happen.
- `Config()` returns an immutable snapshot. A reload never modifies a snapshot you
  already hold; it installs a new one. Read from a snapshot freely, but do not
  call `Set` on it and do not modify maps or slices returned by `GetMap` /
  `GetList`.
- `OnReload` and `OnError` run on the watcher goroutine outside the lock, so they
  may read from the watcher. Do not call `Stop` from inside a callback.
- `Stop` is idempotent, waits for the watcher goroutine to finish, and the last
  config stays readable afterwards.

### Errors never contain file contents

Parse errors passed to `OnError` (and returned by `Watch` for the initial load)
are `*DMLError` values that keep `Type`, `Line` and `Column` but carry a fixed
message and no source line, so values from the file never reach your logs. Open
the file at the reported position to see the problem.

### Example

```go
package main

import (
    "log"
    "net/http"
    "time"

    "github.com/tree-software-company/dml-go/dml"
)

func main() {
    watcher, err := dml.Watch("config.dml", dml.WatchOptions{
        Interval: 2 * time.Second,
        OnReload: func(cfg *dml.Config) {
            log.Printf("config reloaded, %d keys", len(cfg.Keys()))
        },
        OnError: func(err error) {
            log.Printf("config reload failed: %v", err)
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer watcher.Stop()

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(watcher.GetString("app_version")))
    })
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

A runnable version lives in `examples/watch/main.go`.

---

## 🧬 Typed Unmarshaling

Bind a parsed config straight into a Go struct. The target type is a generic type parameter, so the result is typed at compile time and no `interface{}` pointer or type assertion is needed.

### Signatures

```go
func Unmarshal[T any](data []byte) (*T, error)
func UnmarshalFile[T any](path string) (*T, error)
func Decode[T any](c *Config) (*T, error)
```

Go methods cannot have type parameters, so these are package-level functions and the type argument is always written explicitly: `dml.Unmarshal[AppConfig](data)`.

- `Unmarshal` parses DML bytes (up to 10 MiB) and binds the result into a new `T`.
- `UnmarshalFile` does the same for a file. Path validation, the size limit and error redaction match `LoadFiles`.
- `Decode` binds an existing `*Config`, for example after `LoadFiles`, `LoadWithEnv()` or `SetFromString`. Environment expansion is never implicit, call `LoadWithEnv()` before `Decode` when you want it.
- On any error the returned pointer is `nil`, never a half-filled struct.

### Example

```go
type Database struct {
	Host string `dml:"host"`
	Port int    `dml:"port"`
}

type AppConfig struct {
	Name     string   `dml:"name,required"`
	Tags     []string `dml:"tags"`
	Database Database `dml:"database"`
}

data := []byte(`
string name = "BeautyApp";
list tags = ["api", "web"];
map database = {"host": "localhost", "port": 5432};
`)

cfg, err := dml.Unmarshal[AppConfig](data)
if err != nil {
	log.Fatal(err)
}

fmt.Println(cfg.Database.Port)
```

A runnable version lives in `examples/unmarshal/main.go`.

### Behaviour

| Go field type                              | Accepted DML values                                                                  |
| ------------------------------------------ | ------------------------------------------------------------------------------------ |
| `string`, `bool`                           | exactly a string, exactly a boolean                                                  |
| `int`, `int8` ... `int64`                  | an integer, or a float with no fraction, as long as it fits the field                |
| `uint`, `uint8` ... `uint64`               | a non-negative integer, or a float with no fraction, as long as it fits the field    |
| `float32`, `float64`                       | a float, or an integer that converts without losing precision                        |
| struct                                     | a map                                                                                |
| `*T`                                       | anything `T` accepts, allocated only when the key is present                         |
| slice, array                               | a list, element by element (an array needs exactly as many items)                    |
| `map[string]T`                             | a map, value by value                                                                |
| `any`                                      | a copy of whatever DML produced                                                      |

- A struct field is matched by its `dml:"name"` tag, or by the field name when there is no tag. An exact match wins, otherwise the match ignores case, so `Port` finds `port`.
- `dml:"-"` and unexported fields are skipped. `dml:"name,required"` fails with `ErrMissingRequired` when the key is absent or null.
- A missing key keeps the zero value and an unknown key is ignored. A pointer field tells a missing key (`nil`) from a zero value (pointer to `0`).
- Strings, numbers and booleans are never converted into each other: `"8080"` does not fit an `int` field.
- A fractional or non-finite number into an integer field is `ErrTypeMismatch`. An integral number that is too large is `ErrOutOfRange`.
- `T` may also be `map[string]any`, `any` or a pointer to a struct.
- The result never shares maps or slices with the `Config`.

### Errors

Binding failures are `*BindError` values that carry the path of the field, for example `server.port` or `servers[2].name`, and wrap one of the sentinels listed under [Error types](#-error-types). They never contain the offending value, only Go type names.

```go
_, err := dml.Unmarshal[AppConfig]([]byte(`
string name = "BeautyApp";
string database.port = "not a number";
`))

var bindErr *dml.BindError
if errors.As(err, &bindErr) {
	fmt.Println(bindErr.Path, errors.Is(err, dml.ErrTypeMismatch))
}
```

Syntax errors in the DML text surface as a redacted `*DMLError`, the same as in `LoadFiles`.

### Limitations

- An embedded struct is bound like an ordinary named field, it is not flattened.
- Not supported: `time.Duration` from a string, `encoding.TextUnmarshaler`, dotted tag names, a strict mode that rejects unknown keys, and default values.
- The DML parser does not produce lists of maps or maps nested inside map literals. Use dotted names such as `int server.db.port = 5432;` to fill nested structs.
- A field of an unsupported type (channel, function, complex number, non-empty interface, map with non-string keys) is reported only when its key is present with a value.
- Data nested deeper than 128 levels fails with `ErrTooDeep`.
- `Decode` reads the `Config`, so do not call it while another goroutine writes to the same `Config`.

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
| `LoadFiles(files ...string)`              | Loads several `.dml` files and merges them, later files win        |
| `Unmarshal[T any](data []byte)`           | Parses DML bytes into a typed `*T`                                 |
| `UnmarshalFile[T any](path string)`       | Loads a `.dml` file into a typed `*T`                              |
| `Decode[T any](c *Config)`                | Binds an existing `Config` into a typed `*T`                       |
| `Cache(file string)`                      | Loads and caches parsed data in memory                             |
| `Reload(file string)`                     | Forces re-parsing and updates the cache for a file                 |
| `ReloadKeys(file string, keys ...string)` | Partially reloads only the given top-level keys in the cache       |
| `ClearCache()`                            | Clears all cached parsed files from memory                         |
| `Watch(file, WatchOptions)`               | Opt-in live reload of a dml file into a thread-safe `Watcher`      |
| `ApplyDefaults(file, defaults, policy)`   | Apply default values with policy control                           |
| `SetMapStyle(style MapStyle)`             | Sets global map dump style (JSON/Flat/Auto)                        |
| `GetMapStyle()`                           | Returns current global map style                                   |

Every function that takes a file path (`Load`, `NewConfig`, `UnmarshalFile`, `Cache`, `Reload`, `ReloadKeys`, `ApplyDefaults`, `Lint`, `LoadEnv`, `LoadEnvIfExists`, `Config.ReloadKeys`, `Config.SaveToFile`) rejects empty paths, paths containing a NUL byte and paths containing a `..` segment (for example `../../etc/passwd`). The returned error wraps `dml.ErrInvalidPath`, so it can be checked with `errors.Is(err, dml.ErrInvalidPath)`.

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
| `Merge(other *Config)`                           | Deep-merges another config into this one, `other` wins           |
| `SetFromString(assignment string)`               | Applies a `key=value` override, typed by the existing key        |
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
| `BindError`           | Unmarshal error with the path of the field        |
| `ErrTypeMismatch`     | Value kind does not match the field type          |
| `ErrOutOfRange`       | Number does not fit the field type                |
| `ErrUnsupportedType`  | Field type cannot be bound                        |
| `ErrMissingRequired`  | Key marked `required` is absent or null           |
| `ErrTooDeep`          | Data nested deeper than the limit                 |
| `ErrNilConfig`        | `Decode` was called with a nil config             |

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

# Run the typed unmarshaling tests
go test ./dml -run 'Unmarshal|Decode|Bind' -v
go test -race ./dml
go test ./dml -run '^$' -bench 'Unmarshal|Decode' -benchmem
go test ./dml -run '^$' -fuzz FuzzDecodeFromJSON -fuzztime 20s

# Generate coverage report
go test ./dml -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run examples
go run ./examples/policy
go run ./examples/mapstyle
go run ./examples/env
go run ./examples/reload_keys
go run ./examples/watch
go run ./examples/unmarshal

# Run error handling demo
go run ./examples/errors

# Test map style functionality
go run ./examples/mapstyle_dump
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
- ✅ Typed unmarshaling (struct tags, numeric ranges, error paths, depth limits, fuzzing)

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
