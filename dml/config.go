package dml

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
)

type MapStyle int

const (
	MapStyleAuto MapStyle = iota
	MapStyleJSON
	MapStyleFlat
)

var (
	globalMapStyle = MapStyleAuto
	configCache    = make(map[string]map[string]any)
	cacheMutex     sync.RWMutex
)

func SetMapStyle(style MapStyle) {
	globalMapStyle = style
}

func GetMapStyle() MapStyle {
	return globalMapStyle
}

func Cache(filepath string) (map[string]any, error) {
	cacheMutex.RLock()
	if cached, exists := configCache[filepath]; exists {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	cfg := New()
	if err := cfg.Parse(string(content)); err != nil {
		return nil, err
	}

	cacheMutex.Lock()
	configCache[filepath] = cfg.data
	cacheMutex.Unlock()

	return cfg.data, nil
}

func Reload(filepath string) (map[string]any, error) {
	cacheMutex.Lock()
	delete(configCache, filepath)
	cacheMutex.Unlock()

	return Cache(filepath)
}

func ClearCache() {
	cacheMutex.Lock()
	configCache = make(map[string]map[string]any)
	cacheMutex.Unlock()
}

func Load(filepath string) (map[string]any, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	cfg := New()
	if err := cfg.Parse(string(content)); err != nil {
		return nil, err
	}

	return cfg.data, nil
}

type ValidationResult struct {
	MissingKeys []string
	WrongTypes  []string
	IsValid     bool
}

type Config struct {
	data        map[string]any
	defaultKeys map[string]bool
	mapStyle    MapStyle
}

func (c *Config) SetMapStyle(style MapStyle) {
	c.mapStyle = style
}

func (c *Config) getEffectiveMapStyle() MapStyle {
	if c.mapStyle != MapStyleAuto {
		return c.mapStyle
	}
	return globalMapStyle
}

func New() *Config {
	return &Config{
		data:        make(map[string]any),
		defaultKeys: make(map[string]bool),
		mapStyle:    MapStyleAuto,
	}
}

func NewConfig(filename string) (*Config, error) {
	parsed, err := Cache(filename)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to parse DML file '%s': %w", filename, err)
	}

	if parsed == nil {
		parsed = make(map[string]any)
	}

	return &Config{
		data:        parsed,
		defaultKeys: make(map[string]bool),
		mapStyle:    MapStyleAuto,
	}, nil
}

func (c *Config) GetString(key string) string {
	if val, ok := c.resolveNestedKey(key); ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (c *Config) GetInt(key string) int {
	if val, ok := c.resolveNestedKey(key); ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}

func (c *Config) GetFloat(key string) float64 {
	if val, ok := c.resolveNestedKey(key); ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		}
	}
	return 0.0
}

func (c *Config) GetNumber(key string) float64 {
	return c.GetFloat(key)
}

func (c *Config) GetBool(key string) bool {
	if val, ok := c.resolveNestedKey(key); ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func (c *Config) GetList(key string) []any {
	if val, ok := c.resolveNestedKey(key); ok {
		if list, ok := val.([]any); ok {
			return list
		}
	}
	return []any{}
}

func (c *Config) GetMap(key string) map[string]any {
	if val, ok := c.resolveNestedKey(key); ok {
		if m, ok := val.(map[string]any); ok {
			return m
		}
	}
	return map[string]any{}
}

func (c *Config) MustString(key string) string {
	if val, ok := c.resolveNestedKey(key); ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	panic(fmt.Sprintf("❌ Missing required string key: '%s'", key))
}

func (c *Config) MustInt(key string) int {
	if val, ok := c.resolveNestedKey(key); ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	panic(fmt.Sprintf("❌ Missing required int key: '%s'", key))
}

func (c *Config) MustFloat(key string) float64 {
	if val, ok := c.resolveNestedKey(key); ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		}
	}
	panic(fmt.Sprintf("❌ Missing required float key: '%s'", key))
}

func (c *Config) MustBool(key string) bool {
	if val, ok := c.resolveNestedKey(key); ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	panic(fmt.Sprintf("❌ Missing required bool key: '%s'", key))
}

func (c *Config) Has(key string) bool {
	_, ok := c.resolveNestedKey(key)
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
	var builder strings.Builder
	style := c.getEffectiveMapStyle()

	if style == MapStyleJSON {
		builder.WriteString("@mapStyle json\n\n")
	} else if style == MapStyleFlat {
		builder.WriteString("@mapStyle flat\n\n")
	}

	c.dumpData(&builder, c.data, "", style)
	return builder.String()
}

func (c *Config) dumpData(builder *strings.Builder, data map[string]any, prefix string, style MapStyle) {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		c.dumpValue(builder, fullKey, data[key], style)
	}
}

func (c *Config) dumpValue(builder *strings.Builder, key string, value any, style MapStyle) {
	switch v := value.(type) {
	case map[string]any:
		c.dumpMap(builder, key, v, style)
	case []any:
		c.dumpArray(builder, key, v)
	default:
		c.dumpScalar(builder, key, value, style)
	}
}

func (c *Config) dumpMap(builder *strings.Builder, key string, m map[string]any, style MapStyle) {
	if style == MapStyleJSON {
		builder.WriteString(fmt.Sprintf("map %s = {\n", key))
		keys := c.sortedKeys(m)
		for i, k := range keys {
			builder.WriteString(fmt.Sprintf("  \"%s\": ", k))
			c.dumpInlineValue(builder, m[k])
			if i < len(keys)-1 {
				builder.WriteString(",")
			}
			builder.WriteString("\n")
		}
		builder.WriteString("};\n\n")
	} else if style == MapStyleFlat {
		c.dumpData(builder, m, key, style)
	} else {
		if len(m) > 3 || c.hasNestedStructures(m) {
			builder.WriteString(fmt.Sprintf("map %s = {\n", key))
			keys := c.sortedKeys(m)
			for i, k := range keys {
				builder.WriteString(fmt.Sprintf("  \"%s\": ", k))
				c.dumpInlineValue(builder, m[k])
				if i < len(keys)-1 {
					builder.WriteString(",")
				}
				builder.WriteString("\n")
			}
			builder.WriteString("};\n\n")
		} else {
			c.dumpData(builder, m, key, style)
		}
	}
}

func (c *Config) dumpArray(builder *strings.Builder, key string, arr []any) {
	builder.WriteString(fmt.Sprintf("list %s = [", key))
	for i, v := range arr {
		if i > 0 {
			builder.WriteString(", ")
		}
		switch val := v.(type) {
		case string:
			builder.WriteString(fmt.Sprintf("\"%s\"", val))
		default:
			builder.WriteString(fmt.Sprintf("%v", val))
		}
	}
	builder.WriteString("];\n\n")
}

func (c *Config) dumpScalar(builder *strings.Builder, key string, value any, style MapStyle) {
	var typeName string
	switch value.(type) {
	case string:
		typeName = "string"
		builder.WriteString(fmt.Sprintf("%s %s = \"%v\";\n", typeName, key, value))
	case float64, int:
		typeName = "number"
		builder.WriteString(fmt.Sprintf("%s %s = %v;\n", typeName, key, value))
	case bool:
		typeName = "boolean"
		builder.WriteString(fmt.Sprintf("%s %s = %v;\n", typeName, key, value))
	default:
		builder.WriteString(fmt.Sprintf("string %s = \"%v\";\n", key, value))
	}
}

func (c *Config) dumpInlineValue(builder *strings.Builder, value any) {
	switch v := value.(type) {
	case string:
		builder.WriteString(fmt.Sprintf("\"%s\"", v))
	case int, int64:
		builder.WriteString(fmt.Sprintf("%d", v))
	case float64:
		builder.WriteString(fmt.Sprintf("%g", v))
	case bool:
		builder.WriteString(fmt.Sprintf("%t", v))
	case map[string]any:
		builder.WriteString("{ ")
		keys := c.sortedKeys(v)
		for i, k := range keys {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(fmt.Sprintf("\"%s\": ", k))
			c.dumpInlineValue(builder, v[k])
		}
		builder.WriteString(" }")
	case []any:
		builder.WriteString("[")
		for i, val := range v {
			if i > 0 {
				builder.WriteString(", ")
			}
			c.dumpInlineValue(builder, val)
		}
		builder.WriteString("]")
	default:
		builder.WriteString(fmt.Sprintf("%v", value))
	}
}

func (c *Config) sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (c *Config) hasNestedStructures(m map[string]any) bool {
	for _, v := range m {
		switch v.(type) {
		case map[string]any, []any:
			return true
		}
	}
	return false
}

func (c *Config) ValidateRequired(keys ...string) error {
	missing := []string{}

	for _, key := range keys {
		if _, ok := c.resolveNestedKey(key); !ok {
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
		val, ok := c.resolveNestedKey(key)
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

func (c *Config) Set(key string, value any) {
	setNestedKey(c.data, key, value)
}

func (c *Config) Delete(key string) {
	if !strings.Contains(key, ".") {
		delete(c.data, key)
		return
	}

	parts := strings.Split(key, ".")
	var current any = c.data

	for i := 0; i < len(parts)-1; i++ {
		if m, ok := current.(map[string]any); ok {
			current = m[parts[i]]
		} else {
			return
		}
	}

	if m, ok := current.(map[string]any); ok {
		delete(m, parts[len(parts)-1])
	}
}

func (c *Config) Merge(other *Config) {
	for k, v := range other.data {
		c.data[k] = v
	}
}

func (c *Config) Clone() *Config {
	newCfg := New()
	data, _ := json.Marshal(c.data)
	json.Unmarshal(data, &newCfg.data)
	return newCfg
}

func (c *Config) get(key string) any {
	val, _ := c.resolveNestedKey(key)
	return val
}
