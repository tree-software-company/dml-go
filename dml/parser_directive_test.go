package dml

import (
	"strings"
	"testing"
)

func TestParseMapStyleDirective(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectedStyle MapStyle
		expectError   bool
	}{
		{
			name: "JSON style directive",
			content: `@mapStyle json

map server = {
  "port": 8080,
  "host": "localhost"
};`,
			expectedStyle: MapStyleJSON,
			expectError:   false,
		},
		{
			name: "Flat style directive",
			content: `@mapStyle flat

string server.port = "8080";
string server.host = "localhost";`,
			expectedStyle: MapStyleFlat,
			expectError:   false,
		},
		{
			name: "Auto style directive",
			content: `@mapStyle auto

string name = "test";`,
			expectedStyle: MapStyleAuto,
			expectError:   false,
		},
		{
			name: "Invalid style value",
			content: `@mapStyle invalid

string name = "test";`,
			expectedStyle: MapStyleAuto,
			expectError:   true,
		},
		{
			name: "Missing style value",
			content: `@mapStyle

string name = "test";`,
			expectedStyle: MapStyleAuto,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := New()
			err := cfg.Parse(tt.content)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if cfg.getEffectiveMapStyle() != tt.expectedStyle {
				t.Errorf("Expected style %v, got %v", tt.expectedStyle, cfg.getEffectiveMapStyle())
			}
		})
	}
}

func TestDirectivePersistsInDump(t *testing.T) {
	cfg := New()
	content := `@mapStyle json

map server = {
  "port": 8080
};`

	err := cfg.Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	dumped := cfg.Dump()

	if !strings.Contains(dumped, "@mapStyle json") {
		t.Errorf("Expected dump to contain @mapStyle json directive, got:\n%s", dumped)
	}
}

func TestUnknownDirective(t *testing.T) {
	cfg := New()
	content := `@unknownDirective value

string name = "test";`

	err := cfg.Parse(content)
	if err == nil {
		t.Error("Expected error for unknown directive")
	}

	if dmlErr, ok := err.(*DMLError); ok {
		if dmlErr.Type != ErrorTypeValidation {
			t.Errorf("Expected validation error, got %s", dmlErr.Type)
		}
	}
}
