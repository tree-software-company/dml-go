package dml

import (
	"os"
	"testing"
)

func TestDefaultPolicy(t *testing.T) {
	tests := []struct {
		name           string
		existing       map[string]any
		defaults       map[string]any
		policy         DefaultPolicy
		expectedValues map[string]any
		expectError    bool
	}{
		{
			name: "OnlyMissing - adds missing values",
			existing: map[string]any{
				"port": 8080,
			},
			defaults: map[string]any{
				"port":    9000,
				"host":    "localhost",
				"timeout": 30,
			},
			policy: DefaultPolicy{
				OnlyMissing: true,
				StrictTypes: false,
			},
			expectedValues: map[string]any{
				"port":    8080,
				"host":    "localhost",
				"timeout": 30,
			},
		},
		{
			name: "Override - overwrites existing values",
			existing: map[string]any{
				"port": 8080,
			},
			defaults: map[string]any{
				"port": 9000,
			},
			policy: DefaultPolicy{
				Override: true,
			},
			expectedValues: map[string]any{
				"port": 9000,
			},
		},
		{
			name: "StrictTypes - enforces type matching",
			existing: map[string]any{
				"port": 8080,
			},
			defaults: map[string]any{
				"port": "9000",
			},
			policy: DefaultPolicy{
				StrictTypes: true,
				Override:    true,
			},
			expectError: true,
		},
		{
			name: "SkipIfPresent - skips when values exist",
			existing: map[string]any{
				"port": 8080,
			},
			defaults: map[string]any{
				"host": "localhost",
			},
			policy: DefaultPolicy{
				SkipIfPresent: true,
			},
			expectedValues: map[string]any{
				"port": 8080,
			},
		},
		{
			name:     "Permissive policy - allows all",
			existing: map[string]any{},
			defaults: map[string]any{
				"port":    9000,
				"host":    "localhost",
				"timeout": 30,
			},
			policy: DefaultPolicyPermissive,
			expectedValues: map[string]any{
				"port":    9000,
				"host":    "localhost",
				"timeout": 30,
			},
		},
		{
			name: "Strict policy - only missing with types",
			existing: map[string]any{
				"port": 8080,
			},
			defaults: map[string]any{
				"port": 9000,
				"host": "localhost",
			},
			policy: DefaultPolicyStrict,
			expectedValues: map[string]any{
				"port": 8080,
				"host": "localhost",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := New()
			for k, v := range tt.existing {
				cfg.Set(k, v)
			}

			for k, v := range tt.defaults {
				err := cfg.applyDefault(k, v, tt.policy)
				if tt.expectError {
					if err == nil {
						t.Errorf("Expected error but got none")
					}
					return
				}
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			if !tt.expectError {
				for k, expectedVal := range tt.expectedValues {
					actualVal, exists := cfg.Get(k)
					if !exists {
						t.Errorf("Expected key %s to exist", k)
						continue
					}
					if actualVal != expectedVal {
						t.Errorf("For key %s: expected %v, got %v", k, expectedVal, actualVal)
					}
				}
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	tmpFile := "test_policy.dml"
	defer os.Remove(tmpFile)

	initialContent := `number port = 8080;
string host = "localhost";`

	err := os.WriteFile(tmpFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	defaults := map[string]any{
		"port":    9000,
		"timeout": 30,
		"debug":   true,
	}

	err = ApplyDefaults(tmpFile, defaults, DefaultPolicy{
		OnlyMissing: true,
		StrictTypes: false,
	})

	if err != nil {
		t.Fatalf("ApplyDefaults failed: %v", err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load updated file: %v", err)
	}

	config := &Config{data: cfg}

	if config.GetInt("port") != 8080 {
		t.Errorf("Expected port to remain 8080, got %d", config.GetInt("port"))
	}

	if config.GetInt("timeout") != 30 {
		t.Errorf("Expected timeout to be 30, got %d", config.GetInt("timeout"))
	}

	if !config.GetBool("debug") {
		t.Error("Expected debug to be true")
	}
}
