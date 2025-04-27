package dml

import (
    "encoding/json"
    "fmt"
    "sort"
	"reflect"
	"strings"
)

type Config struct {
    data map[string]any
}

func NewConfig(filename string) (*Config, error) {
    parsed, err := Cache(filename)
    if err != nil {
        return nil, fmt.Errorf("❌ Failed to parse DML file '%s': %w", filename, err)
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
    if val, ok := c.data[key]; ok {
        if num, ok := val.(float64); ok {
            return num
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
    bytes, err := json.MarshalIndent(c.data, "", "  ")
    if err != nil {
        return "{}"
    }
    return string(bytes)
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
    current := c.data

    for i, part := range parts {
        value, ok := current[part]
        if !ok {
            return nil, false
        }

        if i == len(parts)-1 {
            return value, true
        }

        nested, ok := value.(map[string]any)
        if !ok {
            return nil, false
        }

        current = nested
    }

    return nil, false
}
