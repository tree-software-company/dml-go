package dml

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePath(t *testing.T) {
	rejected := []string{
		"../../etc/passwd",
		"..",
		"a/../b",
		`a\..\b`,
		"./../x",
		"/abs/../x",
		"config.dml/..",
		"",
		"   ",
		"a\x00b",
	}
	for _, path := range rejected {
		if err := validatePath(path); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("validatePath(%q) = %v, want ErrInvalidPath", path, err)
		}
	}

	accepted := []string{
		"config.dml",
		"./config.dml",
		"/tmp/x/config.dml",
		"..hidden.dml",
		"file..dml",
		"a/.../b",
		"dir/.config",
	}
	for _, path := range accepted {
		if err := validatePath(path); err != nil {
			t.Errorf("validatePath(%q) = %v, want nil", path, err)
		}
	}
}

func traversalPath(dir string) string {
	sep := string(filepath.Separator)
	return filepath.Join(dir, "sub") + sep + ".." + sep + "escaped.dml"
}

func TestEntryPointsRejectTraversal(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "escaped.dml"), []byte(`string host = "localhost";`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	path := traversalPath(dir)

	calls := map[string]func() error{
		"NewConfig": func() error {
			_, err := NewConfig(path)
			return err
		},
		"Load": func() error {
			_, err := Load(path)
			return err
		},
		"Cache": func() error {
			_, err := Cache(path)
			return err
		},
		"Reload": func() error {
			_, err := Reload(path)
			return err
		},
		"ReloadKeys": func() error {
			_, err := ReloadKeys(path, "host")
			return err
		},
		"Config.ReloadKeys": func() error {
			return New().ReloadKeys(path, "host")
		},
		"ApplyDefaults": func() error {
			return ApplyDefaults(path, map[string]any{"timeout": 30}, DefaultPolicyPermissive)
		},
		"Lint": func() error {
			_, err := Lint(path)
			return err
		},
		"LoadEnv": func() error {
			return LoadEnv(path)
		},
		"LoadEnvIfExists": func() error {
			return LoadEnvIfExists(path)
		},
		"SaveToFile": func() error {
			return New().SaveToFile(path)
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, ErrInvalidPath) {
				t.Fatalf("%s returned %v, want ErrInvalidPath", name, err)
			}
		})
	}
}

func TestSaveToFileTraversalDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	cfg := New()
	cfg.Set("host", "localhost")

	if err := cfg.SaveToFile(traversalPath(dir)); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("SaveToFile returned %v, want ErrInvalidPath", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "escaped.dml")); !os.IsNotExist(err) {
		t.Fatalf("escaped.dml must not exist, Stat returned %v", err)
	}
}

func TestSaveToFileAndNewConfigAcceptCleanPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clean.dml")

	cfg := New()
	cfg.Set("host", "localhost")
	if err := cfg.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	loaded, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}
	if got := loaded.GetString("host"); got != "localhost" {
		t.Fatalf("host = %q, want localhost", got)
	}
}
