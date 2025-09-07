package dml

import (
    "fmt"
    "sort"
	"reflect"
	"strings"
    "os"
)

type Config struct {
	data        map[string]any
	defaultKeys map[string]bool
}

type ValidationResult struct {
    MissingKeys []string
    WrongTypes  []string
    IsValid     bool
}

func NewConfig(filename string) (*Config, error) {
    parsed, err := Cache(filename)
    if err != nil {
        return nil, fmt.Errorf("❌ Failed to parse DML file '%s': %w", filename, err)
    }

    if parsed == nil {
        parsed = make(map[string]any)
    }

    return &Config{data: parsed}, nil
}

func (c *Config) GetString(key string) string {
    if val, ok := c.data[key]; ok {
        if str, ok := val.(string); ok {
            return str
        }
    }
    return ""
}

func (c *Config) GetNumber(key string) float64 {
    if val, ok := c.resolveNestedKey(key); ok {
        if f, ok := val.(float64); ok {
            return f
        }
    }
    return 0
}


func (c *Config) GetBool(key string) bool {
    if val, ok := c.data[key]; ok {
        if b, ok := val.(bool); ok {
            return b
        }
    }
    return false
}

func (c *Config) GetList(key string) []any {
    if val, ok := c.data[key]; ok {
        if list, ok := val.([]any); ok {
            return list
        }
    }
    return []any{}
}

func (c *Config) GetMap(key string) map[string]any {
    if val, ok := c.data[key]; ok {
        if m, ok := val.(map[string]any); ok {
            return m
        }
    }
    return map[string]any{}
}

func (c *Config) MustString(key string) string {
    if val, ok := c.data[key]; ok {
        if str, ok := val.(string); ok {
            return str
        }
    }
    panic(fmt.Sprintf("❌ Missing required string key: '%s'", key))
}

func (c *Config) Has(key string) bool {
    _, ok := c.data[key]
    return ok
}

func (c *Config) Keys() []string {
    keys := make([]string, 0, len(c.data))
    for k := range c.data {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    return keys
}

func (c *Config) Dump() string {
	return renderAsDML(c.data, 0)
}

func renderAsDML(data map[string]any, indent int) string {
	ind := strings.Repeat("  ", indent)
	var out strings.Builder

	for k, v := range data {
		switch val := v.(type) {
		case map[string]any:
			out.WriteString(fmt.Sprintf("%smap %s = {\n", ind, k))
            keys := make([]string, 0, len(val))
            for k := range val {
                keys = append(keys, k)
            }
            sort.Strings(keys)

            for i, subk := range keys {
                subv := val[subk]
                comma := ""
                if i < len(keys)-1 {
                    comma = ","
                }

                switch subv := subv.(type) {
                case string:
                    out.WriteString(fmt.Sprintf("%s  \"%s\": \"%s\"%s\n", ind, subk, subv, comma))
                case float64, int:
                    out.WriteString(fmt.Sprintf("%s  \"%s\": %v%s\n", ind, subk, subv, comma))
                case bool:
                    out.WriteString(fmt.Sprintf("%s  \"%s\": %v%s\n", ind, subk, subv, comma))
                default:
                    out.WriteString(fmt.Sprintf("%s  \"%s\": \"%v\"%s\n", ind, subk, subv, comma))
                }
            }

            out.WriteString(fmt.Sprintf("%s};\n\n", ind))
		case string:
			out.WriteString(fmt.Sprintf("%sstring %s = \"%s\";\n", ind, k, val))
		case float64:
			out.WriteString(fmt.Sprintf("%snumber %s = %v;\n", ind, k, val))
		case bool:
			out.WriteString(fmt.Sprintf("%sboolean %s = %v;\n", ind, k, val))
		case []any:
			out.WriteString(fmt.Sprintf("%slist %s = [", ind, k))
			for i, item := range val {
				if i > 0 {
					out.WriteString(", ")
				}
				switch item := item.(type) {
				case string:
					out.WriteString(fmt.Sprintf("\"%s\"", item))
				default:
					out.WriteString(fmt.Sprintf("%v", item))
				}
			}
			out.WriteString("];\n")
		default:
			out.WriteString(fmt.Sprintf("%s %s = %v;\n", ind, k, val))
		}
	}

	return out.String()
}

func (c *Config) ValidateRequired(keys ...string) error {
    missing := []string{}

    for _, key := range keys {
        if _, ok := c.data[key]; !ok {
            missing = append(missing, key)
        }
    }

    if len(missing) > 0 {
        return fmt.Errorf("❌ Missing required keys: %v", missing)
    }
    return nil
}

func (c *Config) ValidateRequiredTyped(rules map[string]string) error {
    missing := []string{}
    wrongType := []string{}

    for key, expectedType := range rules {
        val, ok := c.data[key]
        if !ok {
            missing = append(missing, key)
            continue
        }

        actualType := reflect.TypeOf(val).String()
        if actualType != expectedType {
            wrongType = append(wrongType, fmt.Sprintf("%s (expected %s, got %s)", key, expectedType, actualType))
        }
    }

    if len(missing) > 0 || len(wrongType) > 0 {
        msg := "❌ Validation failed:\n"
        if len(missing) > 0 {
            msg += fmt.Sprintf("  - Missing keys: %v\n", missing)
        }
        if len(wrongType) > 0 {
            msg += fmt.Sprintf("  - Wrong types:\n    %s\n", formatList(wrongType))
        }
        return fmt.Errorf(msg)
    }

    return nil
}

func formatList(list []string) string {
    return "    " + fmt.Sprintf("%s", joinWithNewlines(list))
}

func joinWithNewlines(list []string) string {
    result := ""
    for i, v := range list {
        if i > 0 {
            result += "\n    "
        }
        result += v
    }
    return result
}

func (c *Config) resolveNestedKey(key string) (any, bool) {
	parts := strings.Split(key, ".")
	var current any = c.data

	for _, part := range parts {
		if m, ok := current.(map[string]any); ok {
			current, ok = m[part]
			if !ok {
				return nil, false
			}
		} else {
			return nil, false
		}
	}
	return current, true
}


func (c *Config) MissedKeys(required []string) []string {
    var missing []string
    for _, key := range required {
        if _, ok := c.resolveNestedKey(key); !ok {
            missing = append(missing, key)
        }
    }
    return missing
}

func (c *Config) MissedTypedKeys(expected map[string]string) []string {
    var wrong []string

    for key, wantType := range expected {
        val, ok := c.resolveNestedKey(key)
        if !ok {
            continue
        }

        actualType := fmt.Sprintf("%T", val)
        if actualType != wantType {
            wrong = append(wrong, key)
        }
    }

    return wrong
}

func (c *Config) ValidateState(requiredKeys []string, expectedTypes map[string]string) ValidationResult {
    missing := c.MissedKeys(requiredKeys)
    wrongTypes := c.MissedTypedKeys(expectedTypes)

    return ValidationResult{
        MissingKeys: missing,
        WrongTypes:  wrongTypes,
        IsValid:     len(missing) == 0 && len(wrongTypes) == 0,
    }
}

func SetDefaultsToFile(file string, defaults map[string]any, forceOverride bool) error {
	cfg, err := NewConfig(file)

	if err != nil {
		fmt.Printf("⚠️ Creating new config (could not read '%s'): %v\n", file, err)
		cfg = &Config{data: make(map[string]any)}
	} else if cfg.data == nil {
		cfg.data = make(map[string]any)
	}

	var defaultKeys []string
	for key, defValue := range defaults {
		if forceOverride {
			setNestedKey(cfg.data, key, defValue)
			defaultKeys = append(defaultKeys, key)
		} else {
			val, exists := cfg.resolveNestedKey(key)
			if !exists || isZero(val) {
				setNestedKey(cfg.data, key, defValue)
				defaultKeys = append(defaultKeys, key)
			}

		}
	}

	cfg.SetMetaDefaults(defaultKeys)

	output := cfg.Dump()
	err = os.WriteFile(file, []byte(output), 0644)
	if err != nil {
		return fmt.Errorf("❌ Failed to write file: %w", err)
	}

	fmt.Println("✅ Defaults applied and saved to", file)
	return nil
}

func (c *Config) SetMetaDefaults(keys []string) {
	c.defaultKeys = make(map[string]bool)
	for _, k := range keys {
		c.defaultKeys[k] = true
	}
}


func (c *Config) SetMetaDefaults(keys []string) {
	c.defaultKeys = make(map[string]bool)
	for _, k := range keys {
		c.defaultKeys[k] = true
	}
}

func setNestedKey(data map[string]any, key string, value any) {
	parts := strings.Split(key, ".")
	last := parts[len(parts)-1]

	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]

		if _, ok := data[part]; !ok {
			data[part] = map[string]any{}
		}

		next, ok := data[part].(map[string]any)
		if !ok {
			return
		}
		data = next
	}

	data[last] = value
}

func isZero(val any) bool {
	switch v := val.(type) {
	case string:
		return v == ""
	case float64:
		return v == 0
	case bool:
		return !v
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	default:
		return val == nil
	}
}

