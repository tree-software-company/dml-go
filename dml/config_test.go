package dml

import (
	"os"
	"sync"
	"testing"
)

// writeTempDML writes content to a temp file and returns its path.
// The caller is responsible for removing it after the test.
func writeTempDML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.dml")
	if err != nil {
		t.Fatalf("writeTempDML: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writeTempDML write: %v", err)
	}
	f.Close()
	return f.Name()
}

// ---------------------------------------------------------------------------
// Package-level ReloadKeys
// ---------------------------------------------------------------------------

func TestReloadKeys_UpdatesOnlyRequestedKeys(t *testing.T) {
	initial := `
map server = {"host": "localhost", "port": 8080};
map database = {"host": "db-host", "port": 5432};
string app = "v1";
`
	updated := `
map server = {"host": "prod-server", "port": 9090};
map database = {"host": "prod-db", "port": 5433};
string app = "v2";
`
	path := writeTempDML(t, initial)

	data, err := Cache(path)
	if err != nil {
		t.Fatalf("Cache: %v", err)
	}
	if data["app"] != "v1" {
		t.Fatalf("expected app=v1, got %v", data["app"])
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result, err := ReloadKeys(path, "server")
	if err != nil {
		t.Fatalf("ReloadKeys: %v", err)
	}

	srv, ok := result["server"].(map[string]any)
	if !ok {
		t.Fatalf("server key missing or wrong type")
	}
	if srv["host"] != "prod-server" {
		t.Errorf("expected server.host=prod-server, got %v", srv["host"])
	}

	db, ok := result["database"].(map[string]any)
	if !ok {
		t.Fatalf("database key missing or wrong type")
	}
	if db["host"] != "db-host" {
		t.Errorf("expected database.host=db-host (unchanged), got %v", db["host"])
	}

	if result["app"] != "v1" {
		t.Errorf("expected app=v1 (unchanged), got %v", result["app"])
	}
}

func TestReloadKeys_AbsentKeyInFileIsSkipped(t *testing.T) {
	content := `
map server = {"host": "localhost"};
`
	path := writeTempDML(t, content)

	if _, err := Cache(path); err != nil {
		t.Fatalf("Cache: %v", err)
	}

	result, err := ReloadKeys(path, "missing")
	if err != nil {
		t.Fatalf("ReloadKeys: %v", err)
	}

	if _, exists := result["missing"]; exists {
		t.Error("expected 'missing' to not be present in cache after ReloadKeys")
	}
}

func TestReloadKeys_CreatesCacheEntryIfAbsent(t *testing.T) {
	content := `
map server = {"host": "fresh-host"};
string env = "prod";
`
	path := writeTempDML(t, content)

	ClearCache()

	result, err := ReloadKeys(path, "server", "env")
	if err != nil {
		t.Fatalf("ReloadKeys on empty cache: %v", err)
	}

	srv, ok := result["server"].(map[string]any)
	if !ok {
		t.Fatalf("server missing from result")
	}
	if srv["host"] != "fresh-host" {
		t.Errorf("expected server.host=fresh-host, got %v", srv["host"])
	}
	if result["env"] != "prod" {
		t.Errorf("expected env=prod, got %v", result["env"])
	}
}

func TestReloadKeys_MultipleKeys(t *testing.T) {
	initial := `
map server = {"host": "old-server"};
map database = {"host": "old-db"};
string app = "old-app";
`
	updated := `
map server = {"host": "new-server"};
map database = {"host": "new-db"};
string app = "new-app";
`
	path := writeTempDML(t, initial)
	if _, err := Cache(path); err != nil {
		t.Fatalf("Cache: %v", err)
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result, err := ReloadKeys(path, "server", "database")
	if err != nil {
		t.Fatalf("ReloadKeys: %v", err)
	}

	srv := result["server"].(map[string]any)
	db := result["database"].(map[string]any)

	if srv["host"] != "new-server" {
		t.Errorf("expected server.host=new-server, got %v", srv["host"])
	}
	if db["host"] != "new-db" {
		t.Errorf("expected database.host=new-db, got %v", db["host"])
	}
	if result["app"] != "old-app" {
		t.Errorf("expected app=old-app (unchanged), got %v", result["app"])
	}
}

func TestReloadKeys_ConcurrentSafety(t *testing.T) {
	content := `map server = {"host": "concurrent-host"};`
	path := writeTempDML(t, content)

	if _, err := Cache(path); err != nil {
		t.Fatalf("Cache: %v", err)
	}

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if _, err := ReloadKeys(path, "server"); err != nil {
				t.Errorf("ReloadKeys goroutine error: %v", err)
			}
		}()
	}

	wg.Wait()
}


func TestConfigReloadKeys_UpdatesOnlyRequestedKeys(t *testing.T) {
	initial := `
map server = {"host": "old-host", "port": 8080};
map database = {"host": "old-db"};
string version = "1.0";
`
	updated := `
map server = {"host": "new-host", "port": 9090};
map database = {"host": "new-db"};
string version = "2.0";
`
	path := writeTempDML(t, initial)

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := cfg.ReloadKeys(path, "server"); err != nil {
		t.Fatalf("cfg.ReloadKeys: %v", err)
	}

	srv := cfg.GetMap("server")
	if srv["host"] != "new-host" {
		t.Errorf("expected server.host=new-host, got %v", srv["host"])
	}

	db := cfg.GetMap("database")
	if db["host"] != "old-db" {
		t.Errorf("expected database.host=old-db (unchanged), got %v", db["host"])
	}

	if cfg.GetString("version") != "1.0" {
		t.Errorf("expected version=1.0 (unchanged), got %s", cfg.GetString("version"))
	}
}

func TestConfigReloadKeys_AbsentKeyInFileIsSkipped(t *testing.T) {
	initial := `
map server = {"host": "host-a"};
string extra = "keep-me";
`
	path := writeTempDML(t, initial)

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}

	if err := cfg.ReloadKeys(path, "nonexistent"); err != nil {
		t.Fatalf("cfg.ReloadKeys: %v", err)
	}

	if !cfg.Has("server") {
		t.Error("expected 'server' to still be present")
	}
	if cfg.GetString("extra") != "keep-me" {
		t.Errorf("expected extra=keep-me, got %s", cfg.GetString("extra"))
	}
}

func TestConfigReloadKeys_MultipleKeys(t *testing.T) {
	initial := `
map server = {"host": "s1"};
map database = {"host": "d1"};
string app = "a1";
`
	updated := `
map server = {"host": "s2"};
map database = {"host": "d2"};
string app = "a2";
`
	path := writeTempDML(t, initial)

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := cfg.ReloadKeys(path, "server", "database"); err != nil {
		t.Fatalf("cfg.ReloadKeys: %v", err)
	}

	if cfg.GetMap("server")["host"] != "s2" {
		t.Errorf("expected server.host=s2, got %v", cfg.GetMap("server")["host"])
	}
	if cfg.GetMap("database")["host"] != "d2" {
		t.Errorf("expected database.host=d2, got %v", cfg.GetMap("database")["host"])
	}

	if cfg.GetString("app") != "a1" {
		t.Errorf("expected app=a1 (unchanged), got %s", cfg.GetString("app"))
	}
}
