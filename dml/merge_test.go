package dml

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeLayer(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func configWith(data map[string]any) *Config {
	cfg := New()
	cfg.data = data
	return cfg
}

func TestMergeScalarsAndKeys(t *testing.T) {
	base := configWith(map[string]any{"port": 8080, "name": "base", "keep": "me"})
	overlay := configWith(map[string]any{"port": 9000, "extra": true})

	base.Merge(overlay)

	if base.GetInt("port") != 9000 {
		t.Errorf("port = %d, want 9000", base.GetInt("port"))
	}
	if base.GetString("name") != "base" {
		t.Errorf("name = %q, want base", base.GetString("name"))
	}
	if base.GetString("keep") != "me" {
		t.Errorf("keep = %q, want me", base.GetString("keep"))
	}
	if !base.GetBool("extra") {
		t.Error("extra should be added from the overlay")
	}
}

func TestMergeNestedMapsKeepSiblings(t *testing.T) {
	base := configWith(map[string]any{
		"server": map[string]any{"name": "app", "port": 8080, "limits": map[string]any{"cpu": 1, "mem": 512}},
	})
	overlay := configWith(map[string]any{
		"server": map[string]any{"port": 443, "limits": map[string]any{"mem": 1024}},
	})

	base.Merge(overlay)

	want := map[string]any{
		"name":   "app",
		"port":   443,
		"limits": map[string]any{"cpu": 1, "mem": 1024},
	}
	if got := base.GetMap("server"); !reflect.DeepEqual(got, want) {
		t.Errorf("server = %v, want %v", got, want)
	}
}

func TestMergeReplacesLists(t *testing.T) {
	base := configWith(map[string]any{"hosts": []any{"a", "b", "c"}})
	overlay := configWith(map[string]any{"hosts": []any{"z"}})

	base.Merge(overlay)

	want := []any{"z"}
	if got := base.GetList("hosts"); !reflect.DeepEqual(got, want) {
		t.Errorf("hosts = %v, want %v", got, want)
	}
}

func TestMergeOverlayChangesType(t *testing.T) {
	base := configWith(map[string]any{"port": 8080, "server": map[string]any{"a": 1}})
	overlay := configWith(map[string]any{"port": "dynamic", "server": "flat"})

	base.Merge(overlay)

	if base.GetString("port") != "dynamic" {
		t.Errorf("port = %v, want dynamic", base.data["port"])
	}
	if base.GetString("server") != "flat" {
		t.Errorf("server = %v, want flat", base.data["server"])
	}
}

func TestMergeDoesNotAliasOverlay(t *testing.T) {
	overlay := configWith(map[string]any{
		"server": map[string]any{"port": 1},
		"hosts":  []any{"a"},
	})
	base := New()

	base.Merge(overlay)
	overlay.data["server"].(map[string]any)["port"] = 2
	overlay.data["hosts"].([]any)[0] = "changed"

	if got := base.GetInt("server.port"); got != 1 {
		t.Errorf("server.port = %d, want 1", got)
	}
	if got := base.GetList("hosts"); got[0] != "a" {
		t.Errorf("hosts[0] = %v, want a", got[0])
	}
}

func TestMergeMapStyleOnlyWhenExplicit(t *testing.T) {
	base := New()
	base.SetMapStyle(MapStyleFlat)

	base.Merge(New())
	if base.mapStyle != MapStyleFlat {
		t.Errorf("mapStyle = %v, want flat after merging an auto overlay", base.mapStyle)
	}

	overlay := New()
	overlay.SetMapStyle(MapStyleJSON)
	base.Merge(overlay)
	if base.mapStyle != MapStyleJSON {
		t.Errorf("mapStyle = %v, want json after merging an explicit overlay", base.mapStyle)
	}
}

func TestMergeNilOverlay(t *testing.T) {
	base := configWith(map[string]any{"a": 1})

	base.Merge(nil)

	if base.GetInt("a") != 1 {
		t.Error("merging nil must leave the config untouched")
	}
}

func TestLoadFilesBaseAndProd(t *testing.T) {
	cfg, err := LoadFiles(fixture(t, "config.base.dml"), fixture(t, "config.prod.dml"))
	if err != nil {
		t.Fatalf("LoadFiles: %v", err)
	}

	if cfg.GetString("app_name") != "tree" {
		t.Errorf("app_name = %q, want tree", cfg.GetString("app_name"))
	}
	if cfg.GetBool("debug") {
		t.Error("debug should be overridden to false")
	}
	if cfg.GetInt("workers") != 2 {
		t.Errorf("workers = %d, want 2", cfg.GetInt("workers"))
	}
	if cfg.GetString("region") != "eu-west-1" {
		t.Errorf("region = %q, want eu-west-1", cfg.GetString("region"))
	}
	if got := cfg.GetList("allowed_hosts"); !reflect.DeepEqual(got, []any{"app.example.com"}) {
		t.Errorf("allowed_hosts = %v", got)
	}

	wantServer := map[string]any{"name": "BeautyApp", "port": 443, "timeout": 15, "tls": true}
	if got := cfg.GetMap("server"); !reflect.DeepEqual(got, wantServer) {
		t.Errorf("server = %v, want %v", got, wantServer)
	}
	wantDatabase := map[string]any{"host": "db.prod.example.com", "port": 5432}
	if got := cfg.GetMap("database"); !reflect.DeepEqual(got, wantDatabase) {
		t.Errorf("database = %v, want %v", got, wantDatabase)
	}
}

func TestLoadFilesSingleFileMatchesNewConfig(t *testing.T) {
	single, err := NewConfig(fixture(t, "config.base.dml"))
	if err != nil {
		t.Fatal(err)
	}
	layered, err := LoadFiles(fixture(t, "config.base.dml"))
	if err != nil {
		t.Fatal(err)
	}

	if single.Dump() != layered.Dump() {
		t.Errorf("single file dump differs:\n%s\nvs\n%s", single.Dump(), layered.Dump())
	}
}

func TestLoadFilesLastFileWins(t *testing.T) {
	dir := t.TempDir()
	first := writeLayer(t, dir, "a.dml", "int port = 1;\nstring only_a = \"a\";\n")
	second := writeLayer(t, dir, "b.dml", "int port = 2;\nstring only_b = \"b\";\n")
	third := writeLayer(t, dir, "c.dml", "int port = 3;\n")

	cfg, err := LoadFiles(first, second, third)
	if err != nil {
		t.Fatalf("LoadFiles: %v", err)
	}

	if cfg.GetInt("port") != 3 {
		t.Errorf("port = %d, want 3", cfg.GetInt("port"))
	}
	if cfg.GetString("only_a") != "a" || cfg.GetString("only_b") != "b" {
		t.Error("keys unique to earlier files must survive")
	}
}

func TestLoadFilesNoPaths(t *testing.T) {
	if _, err := LoadFiles(); !errors.Is(err, ErrNoLayers) {
		t.Errorf("err = %v, want ErrNoLayers", err)
	}
}

func TestLoadFilesMissingFile(t *testing.T) {
	dir := t.TempDir()
	present := writeLayer(t, dir, "a.dml", "int port = 1;\n")

	_, err := LoadFiles(present, filepath.Join(dir, "missing.dml"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v, want os.ErrNotExist", err)
	}
}

func TestLoadFilesRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	present := writeLayer(t, dir, "a.dml", "int port = 1;\n")

	_, err := LoadFiles(present, "../secret.dml")
	if !errors.Is(err, ErrInvalidPath) {
		t.Errorf("err = %v, want ErrInvalidPath", err)
	}
}

func TestLoadFilesParseErrorIsRedacted(t *testing.T) {
	dir := t.TempDir()
	first := writeLayer(t, dir, "a.dml", "int port = 1;\n")
	second := writeLayer(t, dir, "b.dml", "int port = 2;\nint broken = swordfish;\n")

	_, err := LoadFiles(first, second)

	var dmlErr *DMLError
	if !errors.As(err, &dmlErr) {
		t.Fatalf("err = %v, want a *DMLError", err)
	}
	if dmlErr.Line != 2 {
		t.Errorf("line = %d, want 2", dmlErr.Line)
	}
	if strings.Contains(err.Error(), "swordfish") {
		t.Errorf("error leaks file content: %v", err)
	}
	if !strings.Contains(err.Error(), "file 2") {
		t.Errorf("error should name the failing file position: %v", err)
	}
}

func TestLoadFilesRejectsOversizedFile(t *testing.T) {
	dir := t.TempDir()
	big := writeLayer(t, dir, "big.dml", string(bytes.Repeat([]byte("a"), maxLayerFileSize+1)))

	if _, err := LoadFiles(big); !errors.Is(err, ErrLayerTooLarge) {
		t.Errorf("err = %v, want ErrLayerTooLarge", err)
	}
}

func TestSetFromStringCoercesExistingTypes(t *testing.T) {
	cfg := configWith(map[string]any{
		"port":    8080,
		"ratio":   0.5,
		"debug":   true,
		"name":    "app",
		"server":  map[string]any{"port": 1},
		"numbers": []any{1},
	})

	cases := []struct {
		assignment string
		key        string
		want       any
	}{
		{"port=9000", "port", 9000},
		{"ratio=0.75", "ratio", 0.75},
		{"debug=off", "debug", false},
		{"name=other", "name", "other"},
		{"server.port=443", "server.port", 443},
	}
	for _, tc := range cases {
		if err := cfg.SetFromString(tc.assignment); err != nil {
			t.Fatalf("SetFromString(%q): %v", tc.assignment, err)
		}
		got, _ := cfg.Get(tc.key)
		if got != tc.want {
			t.Errorf("%s = %v (%T), want %v (%T)", tc.key, got, got, tc.want, tc.want)
		}
	}
}

func TestSetFromStringInfersNewKeys(t *testing.T) {
	cfg := New()

	cases := []struct {
		assignment string
		key        string
		want       any
	}{
		{"a.b.c=1", "a.b.c", 1},
		{"ratio=1.5", "ratio", 1.5},
		{"flag=TRUE", "flag", true},
		{"name=hello", "name", "hello"},
		{"nan=NaN", "nan", "NaN"},
		{"empty=", "empty", ""},
		{"url=https://x.test/?a=b", "url", "https://x.test/?a=b"},
	}
	for _, tc := range cases {
		if err := cfg.SetFromString(tc.assignment); err != nil {
			t.Fatalf("SetFromString(%q): %v", tc.assignment, err)
		}
		got, _ := cfg.Get(tc.key)
		if got != tc.want {
			t.Errorf("%s = %v (%T), want %v (%T)", tc.key, got, got, tc.want, tc.want)
		}
	}
}

func TestSetFromStringRejectsBadAssignments(t *testing.T) {
	cfg := New()

	for _, assignment := range []string{"", "noequals", "=value", "  =value", "a..b=1", "a.=1", "1abc=2", "a b=2"} {
		if err := cfg.SetFromString(assignment); !errors.Is(err, ErrInvalidAssignment) {
			t.Errorf("SetFromString(%q) = %v, want ErrInvalidAssignment", assignment, err)
		}
	}
	if len(cfg.data) != 0 {
		t.Errorf("rejected assignments must not change the config: %v", cfg.data)
	}
}

func TestSetFromStringTypeMismatchDoesNotEchoValue(t *testing.T) {
	cfg := configWith(map[string]any{
		"port":   8080,
		"ratio":  0.5,
		"debug":  true,
		"server": map[string]any{"port": 1},
		"hosts":  []any{"a"},
	})

	for _, assignment := range []string{
		"port=zzz-secret",
		"ratio=zzz-secret",
		"debug=zzz-secret",
		"server=zzz-secret",
		"hosts=zzz-secret",
	} {
		err := cfg.SetFromString(assignment)
		if !errors.Is(err, ErrInvalidOverride) {
			t.Errorf("SetFromString(%q) = %v, want ErrInvalidOverride", assignment, err)
			continue
		}
		if strings.Contains(err.Error(), "zzz-secret") {
			t.Errorf("error for %q echoes the value: %v", assignment, err)
		}
	}
	if cfg.GetInt("port") != 8080 || !cfg.GetBool("debug") {
		t.Error("failed overrides must not change existing values")
	}
}

func TestLayeredPrecedence(t *testing.T) {
	load := func(t *testing.T) *Config {
		t.Helper()
		cfg, err := LoadFiles(fixture(t, "config.base.dml"), fixture(t, "config.prod.dml"))
		if err != nil {
			t.Fatalf("LoadFiles: %v", err)
		}
		return cfg
	}

	t.Run("prod file overrides base file", func(t *testing.T) {
		if got := load(t).GetInt("server.port"); got != 443 {
			t.Errorf("server.port = %d, want 443", got)
		}
	})

	t.Run("environment overrides files", func(t *testing.T) {
		t.Setenv("APP_SERVER_PORT", "9000")
		cfg := load(t)
		cfg.EnvOverride("APP")
		if got := cfg.GetInt("server.port"); got != 9000 {
			t.Errorf("server.port = %d, want 9000", got)
		}
	})

	t.Run("set overrides environment", func(t *testing.T) {
		t.Setenv("APP_SERVER_PORT", "9000")
		cfg := load(t)
		cfg.EnvOverride("APP")
		if err := cfg.SetFromString("server.port=7000"); err != nil {
			t.Fatal(err)
		}
		if got := cfg.GetInt("server.port"); got != 7000 {
			t.Errorf("server.port = %d, want 7000", got)
		}
	})
}
