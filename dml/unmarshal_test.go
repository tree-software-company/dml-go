package dml

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type serverSection struct {
	Name    string
	Port    int
	Timeout int
}

type databaseSection struct {
	Host string
	Port int
}

type fixtureConfig struct {
	Server   serverSection
	Database databaseSection
}

type databasePort struct {
	Port int
}

type dottedServer struct {
	Name  string
	Port  int
	Debug bool
	Ratio float64
	DB    databasePort `dml:"db"`
}

type dottedConfig struct {
	Server dottedServer
	Tags   []string
}

type layeredConfig struct {
	AppName      string `dml:"app_name"`
	Debug        bool
	Workers      int
	AllowedHosts []string `dml:"allowed_hosts"`
	Region       string
	Server       struct {
		Name string
		Port int
		TLS  bool `dml:"tls"`
	}
	Database databaseSection
}

var (
	_ func([]byte) (*fixtureConfig, error)  = Unmarshal[fixtureConfig]
	_ func(string) (*fixtureConfig, error)  = UnmarshalFile[fixtureConfig]
	_ func(*Config) (*fixtureConfig, error) = Decode[fixtureConfig]
)

func TestUnmarshalFixtureIntoStruct(t *testing.T) {
	content, err := os.ReadFile(fixture(t, "config.dml"))
	if err != nil {
		t.Fatal(err)
	}

	got, err := Unmarshal[fixtureConfig](content)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	want := fixtureConfig{
		Server:   serverSection{Name: "BeautyApp", Port: 8080, Timeout: 15},
		Database: databaseSection{Host: "localhost", Port: 5432},
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("got %+v, want %+v", *got, want)
	}
}

func TestUnmarshalResultIsTypedAtCompileTime(t *testing.T) {
	got, err := Unmarshal[fixtureConfig]([]byte("int server.port = 8080;\n"))
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	var port int = got.Server.Port
	if port != 8080 {
		t.Errorf("port = %d, want 8080", port)
	}
}

func TestUnmarshalDottedNamesIntoNestedStruct(t *testing.T) {
	source := strings.Join([]string{
		`string server.name = "app";`,
		`int server.port = 8080;`,
		`bool server.debug = true;`,
		`float server.ratio = 1.5;`,
		`int server.db.port = 5432;`,
		`list tags = ["a", "b"];`,
	}, "\n")

	got, err := Unmarshal[dottedConfig]([]byte(source))
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	want := dottedConfig{
		Server: dottedServer{Name: "app", Port: 8080, Debug: true, Ratio: 1.5, DB: databasePort{Port: 5432}},
		Tags:   []string{"a", "b"},
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("got %+v, want %+v", *got, want)
	}
}

func TestUnmarshalFileMatchesUnmarshal(t *testing.T) {
	path := fixture(t, "config.dml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	fromBytes, err := Unmarshal[fixtureConfig](content)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	fromFile, err := UnmarshalFile[fixtureConfig](path)
	if err != nil {
		t.Fatalf("UnmarshalFile: %v", err)
	}

	if !reflect.DeepEqual(fromBytes, fromFile) {
		t.Errorf("Unmarshal = %+v, UnmarshalFile = %+v", *fromBytes, *fromFile)
	}
}

func TestDecodeMatchesUnmarshal(t *testing.T) {
	cfg, err := NewConfig(fixture(t, "config.dml"))
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := Decode[fixtureConfig](cfg)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	content, err := os.ReadFile(fixture(t, "config.dml"))
	if err != nil {
		t.Fatal(err)
	}
	unmarshaled, err := Unmarshal[fixtureConfig](content)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if !reflect.DeepEqual(decoded, unmarshaled) {
		t.Errorf("Decode = %+v, Unmarshal = %+v", *decoded, *unmarshaled)
	}
}

func TestDecodeLayeredConfig(t *testing.T) {
	cfg, err := LoadFiles(fixture(t, "config.base.dml"), fixture(t, "config.prod.dml"))
	if err != nil {
		t.Fatalf("LoadFiles: %v", err)
	}

	got, err := Decode[layeredConfig](cfg)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if got.AppName != "tree" || got.Workers != 2 || got.Region != "eu-west-1" {
		t.Errorf("unexpected scalars: %+v", *got)
	}
	if got.Debug {
		t.Error("debug = true, want the prod layer to override it with false")
	}
	if !reflect.DeepEqual(got.AllowedHosts, []string{"app.example.com"}) {
		t.Errorf("allowed hosts = %v", got.AllowedHosts)
	}
	if got.Server.Name != "BeautyApp" || got.Server.Port != 443 || !got.Server.TLS {
		t.Errorf("server = %+v, want the prod port merged into the base server", got.Server)
	}
	if got.Database.Host != "db.prod.example.com" || got.Database.Port != 5432 {
		t.Errorf("database = %+v", got.Database)
	}
}

func TestDecodeAppliesEnvironmentOnlyAfterLoadWithEnv(t *testing.T) {
	t.Setenv("UNMARSHAL_TEST_HOST", "db.internal")

	cfg := New()
	if err := cfg.Parse(`string host = "${UNMARSHAL_TEST_HOST}";`); err != nil {
		t.Fatal(err)
	}

	type hostConfig struct{ Host string }

	raw, err := Decode[hostConfig](cfg)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if raw.Host != "${UNMARSHAL_TEST_HOST}" {
		t.Errorf("host = %q, want the unexpanded reference", raw.Host)
	}

	cfg.LoadWithEnv()
	expanded, err := Decode[hostConfig](cfg)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if expanded.Host != "db.internal" {
		t.Errorf("host = %q, want db.internal", expanded.Host)
	}
}

func TestDecodeAfterSetFromString(t *testing.T) {
	cfg := New()
	if err := cfg.Parse("int server.port = 8080;\n"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetFromString("server.port=9090"); err != nil {
		t.Fatal(err)
	}

	got, err := Decode[fixtureConfig](cfg)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Server.Port != 9090 {
		t.Errorf("port = %d, want 9090", got.Server.Port)
	}
}

func TestUnmarshalInvalidDMLIsRedacted(t *testing.T) {
	_, err := Unmarshal[fixtureConfig]([]byte("int port = 1;\nint broken = swordfish;\n"))

	var dmlErr *DMLError
	if !errors.As(err, &dmlErr) {
		t.Fatalf("err = %v, want a *DMLError", err)
	}
	if dmlErr.Line != 2 {
		t.Errorf("line = %d, want 2", dmlErr.Line)
	}
	if strings.Contains(err.Error(), "swordfish") {
		t.Errorf("error leaks input content: %v", err)
	}
}

func TestUnmarshalRejectsOversizedInput(t *testing.T) {
	_, err := Unmarshal[fixtureConfig](bytes.Repeat([]byte("a"), maxLayerFileSize+1))

	if !errors.Is(err, ErrLayerTooLarge) {
		t.Errorf("err = %v, want ErrLayerTooLarge", err)
	}
}

func TestUnmarshalFileRejectsOversizedFile(t *testing.T) {
	dir := t.TempDir()
	big := writeLayer(t, dir, "big.dml", string(bytes.Repeat([]byte("a"), maxLayerFileSize+1)))

	_, err := UnmarshalFile[fixtureConfig](big)

	if !errors.Is(err, ErrLayerTooLarge) {
		t.Errorf("err = %v, want ErrLayerTooLarge", err)
	}
}

func TestUnmarshalFileRejectsInvalidPaths(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeLayer(t, dir, "escaped.dml", "int port = 1;\n")

	for name, path := range map[string]string{
		"traversal": traversalPath(dir),
		"empty":     "",
		"blank":     "   ",
	} {
		_, err := UnmarshalFile[fixtureConfig](path)
		if !errors.Is(err, ErrInvalidPath) {
			t.Errorf("%s: err = %v, want ErrInvalidPath", name, err)
		}
	}
}

func TestUnmarshalFileMissingFile(t *testing.T) {
	_, err := UnmarshalFile[fixtureConfig](filepath.Join(t.TempDir(), "missing.dml"))

	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v, want os.ErrNotExist", err)
	}
}

func TestUnmarshalFileParseErrorIsRedacted(t *testing.T) {
	path := writeLayer(t, t.TempDir(), "bad.dml", "int broken = swordfish;\n")

	_, err := UnmarshalFile[fixtureConfig](path)

	var dmlErr *DMLError
	if !errors.As(err, &dmlErr) {
		t.Fatalf("err = %v, want a *DMLError", err)
	}
	if strings.Contains(err.Error(), "swordfish") {
		t.Errorf("error leaks file content: %v", err)
	}
}

func TestDecodeNilConfig(t *testing.T) {
	got, err := Decode[fixtureConfig](nil)

	if !errors.Is(err, ErrNilConfig) {
		t.Errorf("err = %v, want ErrNilConfig", err)
	}
	if got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}

func TestDecodeEmptyConfigGivesZeroValue(t *testing.T) {
	got, err := Decode[fixtureConfig](New())
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !reflect.DeepEqual(*got, fixtureConfig{}) {
		t.Errorf("got %+v, want the zero value", *got)
	}
}

func TestUnmarshalErrorReturnsNilResult(t *testing.T) {
	got, err := Unmarshal[fixtureConfig]([]byte(`string server.port = "not a number";`))

	if err == nil {
		t.Fatal("expected an error")
	}
	if got != nil {
		t.Errorf("got %+v, want nil on error", got)
	}
}

func TestUnmarshalIntoGenericRoots(t *testing.T) {
	source := []byte("int server.port = 8080;\nstring name = \"app\";\n")

	t.Run("map", func(t *testing.T) {
		got, err := Unmarshal[map[string]any](source)
		if err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		want := map[string]any{"server": map[string]any{"port": 8080}, "name": "app"}
		if !reflect.DeepEqual(*got, want) {
			t.Errorf("got %v, want %v", *got, want)
		}
	})

	t.Run("any", func(t *testing.T) {
		got, err := Unmarshal[any](source)
		if err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		want := map[string]any{"server": map[string]any{"port": 8080}, "name": "app"}
		if !reflect.DeepEqual(*got, want) {
			t.Errorf("got %v, want %v", *got, want)
		}
	})

	t.Run("pointer to struct", func(t *testing.T) {
		got, err := Unmarshal[*fixtureConfig](source)
		if err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if *got == nil || (*got).Server.Port != 8080 {
			t.Errorf("got %+v, want port 8080", *got)
		}
	})

	t.Run("scalar", func(t *testing.T) {
		_, err := Unmarshal[int](source)
		assertBindError(t, err, ErrTypeMismatch, "root")
	})
}

func TestDecodeDoesNotAliasConfig(t *testing.T) {
	cfg := New()
	if err := cfg.Parse("list tags = [\"a\", \"b\"];\nint server.port = 1;\n"); err != nil {
		t.Fatal(err)
	}

	got, err := Decode[map[string]any](cfg)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	(*got)["tags"].([]any)[0] = "mutated"
	(*got)["server"].(map[string]any)["port"] = 99
	(*got)["added"] = true

	if cfg.GetList("tags")[0] != "a" {
		t.Errorf("tags in config changed: %v", cfg.GetList("tags"))
	}
	if cfg.GetInt("server.port") != 1 {
		t.Errorf("server.port in config changed: %d", cfg.GetInt("server.port"))
	}
	if _, exists := cfg.Get("added"); exists {
		t.Error("key added to the result leaked into the config")
	}
}

func TestUnmarshalConcurrentCalls(t *testing.T) {
	source := []byte("int server.port = 8080;\nstring server.name = \"app\";\n")

	var group sync.WaitGroup
	failures := make(chan string, 16)
	for i := 0; i < 16; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			got, err := Unmarshal[fixtureConfig](source)
			if err != nil || got.Server.Port != 8080 || got.Server.Name != "app" {
				failures <- "unexpected result"
			}
		}()
	}
	group.Wait()
	close(failures)

	for failure := range failures {
		t.Error(failure)
	}
}

func assertBindError(t *testing.T, err error, want error, wantPath string) {
	t.Helper()

	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	var bindErr *BindError
	if !errors.As(err, &bindErr) {
		t.Fatalf("err = %v, want a *BindError", err)
	}
	if bindErr.Path != wantPath {
		t.Errorf("path = %q, want %q", bindErr.Path, wantPath)
	}
}
