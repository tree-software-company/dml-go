package dml

import (
	"os"
	"testing"
)

func runLintFromString(t *testing.T, src string) ([]LintIssue, error) {
	t.Helper()
	tmp, err := os.CreateTemp("", "lint-*.dml")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.WriteString(src); err != nil {
		tmp.Close()
		t.Fatalf("write temp: %v", err)
	}
	tmp.Close()
	issues, err := Lint(name)
	return issues, err
}

func findIssue(issues []LintIssue, code string) *LintIssue {
	for i := range issues {
		if issues[i].Code == code {
			return &issues[i]
		}
	}
	return nil
}

func TestLint_BasicChecks(t *testing.T) {
	t.Run("EmptyMap", func(t *testing.T) {
		src := "myMap = {\n}\n"
		issues, err := runLintFromString(t, src)
		if err != nil {
			t.Fatalf("Lint error: %v", err)
		}
		it := findIssue(issues, "EMPTY_MAP")
		if it == nil {
			t.Fatalf("expected EMPTY_MAP issue")
		}
		if it.Line != 1 {
			t.Errorf("expected EMPTY_MAP line 1, got %d", it.Line)
		}
	})

	t.Run("TypedMapEntry", func(t *testing.T) {
		src := "myMap = {\n    string port = 8080\n}\n"
		issues, err := runLintFromString(t, src)
		if err != nil {
			t.Fatalf("Lint error: %v", err)
		}
		it := findIssue(issues, "TYPED_MAP_ENTRY")
		if it == nil {
			t.Fatalf("expected TYPED_MAP_ENTRY issue")
		}
		if it.Line != 2 {
			t.Errorf("expected TYPED_MAP_ENTRY line 2, got %d", it.Line)
		}
	})

	t.Run("TrailingComma", func(t *testing.T) {
		src := "myMap = {\n    port = 8080,\n}\n"
		issues, err := runLintFromString(t, src)
		if err != nil {
			t.Fatalf("Lint error: %v", err)
		}
		it := findIssue(issues, "MAP_TRAILING_COMMA")
		if it == nil {
			t.Fatalf("expected MAP_TRAILING_COMMA issue")
		}
		if it.Line != 2 {
			t.Errorf("expected MAP_TRAILING_COMMA line 2, got %d", it.Line)
		}
	})

	t.Run("MapUnclosed", func(t *testing.T) {
		src := "myMap = {\n    port = 8080\n"
		issues, err := runLintFromString(t, src)
		if err != nil {
			t.Fatalf("Lint error: %v", err)
		}
		it := findIssue(issues, "MAP_UNCLOSED")
		if it == nil {
			t.Fatalf("expected MAP_UNCLOSED issue")
		}
		if it.Line != 1 {
			t.Errorf("expected MAP_UNCLOSED line 1, got %d", it.Line)
		}
	})

	t.Run("MixedMapStyle", func(t *testing.T) {
		src := "var1 = 1\nmyMap = {\n  port = 8080\n}\n"
		issues, err := runLintFromString(t, src)
		if err != nil {
			t.Fatalf("Lint error: %v", err)
		}
		it := findIssue(issues, "MIXED_MAP_STYLE")
		if it == nil {
			t.Fatalf("expected MIXED_MAP_STYLE issue")
		}
	})

	t.Run("UnusedDefault", func(t *testing.T) {
		src := "default foo = 1\nother = 2\n"
		issues, err := runLintFromString(t, src)
		if err != nil {
			t.Fatalf("Lint error: %v", err)
		}
		it := findIssue(issues, "UNUSED_DEFAULT")
		if it == nil {
			t.Fatalf("expected UNUSED_DEFAULT issue")
		}
		if it.Line != 1 {
			t.Errorf("expected UNUSED_DEFAULT line 1, got %d", it.Line)
		}
	})

	t.Run("DefaultUsed", func(t *testing.T) {
		src := "default foo = 1\nfoo = 2\n"
		issues, err := runLintFromString(t, src)
		if err != nil {
			t.Fatalf("Lint error: %v", err)
		}
		it := findIssue(issues, "UNUSED_DEFAULT")
		if it != nil {
			t.Fatalf("did not expect UNUSED_DEFAULT when default is reassigned")
		}
	})
}
