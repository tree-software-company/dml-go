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

type DefaultPolicy struct {
	Override      bool
	StrictTypes   bool
	OnlyMissing   bool
	SkipIfPresent bool
}

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

var DefaultPolicyPermissive = DefaultPolicy{
	Override:      true,
	StrictTypes:   false,
	OnlyMissing:   false,
	SkipIfPresent: false,
}

var DefaultPolicyStrict = DefaultPolicy{
	Override:      false,
	StrictTypes:   true,
	OnlyMissing:   true,
	SkipIfPresent: false,
}

var DefaultPolicyConservative = DefaultPolicy{
	Override:      false,
	StrictTypes:   true,
	OnlyMissing:   false,
	SkipIfPresent: true,
}

type Config struct {
	data        map[string]any
	defaultKeys map[string]bool
	mapStyle    MapStyle
}

func New() *Config {
	return &Config{
		data:        make(map[string]any),
		defaultKeys: make(map[string]bool),
		mapStyle:    MapStyleAuto,
	}
}

func NewConfig(filepath string) (*Config, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	cfg := New()
	if err := cfg.Parse(string(content)); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) ReloadKeys(filepath string, keys ...string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	fresh := New()
	if err := fresh.Parse(string(content)); err != nil {
		return err
	}

	for _, key := range keys {
		if val, exists := fresh.data[key]; exists {
			c.data[key] = val
		}
	}

	return nil
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

func (c *Config) Set(key string, value any) {
	keys := strings.Split(key, ".")
	current := c.data

	for i := 0; i < len(keys)-1; i++ {
		if _, exists := current[keys[i]]; !exists {
			current[keys[i]] = make(map[string]any)
		}
		if nested, ok := current[keys[i]].(map[string]any); ok {
			current = nested
		} else {
			current[keys[i]] = make(map[string]any)
			current = current[keys[i]].(map[string]any)
		}
	}

	current[keys[len(keys)-1]] = value
}

func (c *Config) Get(key string) (any, bool) {
	keys := strings.Split(key, ".")
	current := c.data

	for i := 0; i < len(keys)-1; i++ {
		if val, exists := current[keys[i]]; exists {
			if nested, ok := val.(map[string]any); ok {
				current = nested
			} else {
				return nil, false
			}
		} else {
			return nil, false
		}
	}

	val, exists := current[keys[len(keys)-1]]
	return val, exists
}

func (c *Config) GetString(key string) string {
	if val, exists := c.Get(key); exists {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func (c *Config) GetInt(key string) int {
	if val, exists := c.Get(key); exists {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			var i int
			fmt.Sscanf(v, "%d", &i)
			return i
		}
	}
	return 0
}

func (c *Config) GetFloat(key string) float64 {
	if val, exists := c.Get(key); exists {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case string:
			var f float64
			fmt.Sscanf(v, "%f", &f)
			return f
		}
	}
	return 0.0
}

func (c *Config) GetNumber(key string) float64 {
	return c.GetFloat(key)
}

func (c *Config) GetBool(key string) bool {
	if val, exists := c.Get(key); exists {
		if b, ok := val.(bool); ok {
			return b
		}
		if str, ok := val.(string); ok {
			return str == "true"
		}
	}
	return false
}

func (c *Config) GetList(key string) []any {
	if val, exists := c.Get(key); exists {
		if list, ok := val.([]any); ok {
			return list
		}
	}
	return []any{}
}

func (c *Config) GetMap(key string) map[string]any {
	if val, exists := c.Get(key); exists {
		if m, ok := val.(map[string]any); ok {
			return m
		}
	}
	return make(map[string]any)
}

func (c *Config) Has(key string) bool {
	_, exists := c.Get(key)
	return exists
}

func (c *Config) Keys() []string {
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (c *Config) MustString(key string) string {
	val := c.GetString(key)
	if val == "" && !c.Has(key) {
		panic(fmt.Sprintf("required key '%s' not found", key))
	}
	return val
}

func (c *Config) ValidateRequired(keys ...string) error {
	for _, key := range keys {
		if !c.Has(key) {
			return fmt.Errorf("required key '%s' not found", key)
		}
	}
	return nil
}

func (c *Config) ValidateRequiredTyped(rules map[string]string) error {
	for key, expectedType := range rules {
		if !c.Has(key) {
			return fmt.Errorf("required key '%s' not found", key)
		}

		val, _ := c.Get(key)
		actualType := fmt.Sprintf("%T", val)

		typeMatches := false
		switch expectedType {
		case "string":
			_, typeMatches = val.(string)
		case "int":
			_, ok1 := val.(int)
			_, ok2 := val.(float64)
			typeMatches = ok1 || ok2
		case "float":
			_, ok1 := val.(float64)
			_, ok2 := val.(int)
			typeMatches = ok1 || ok2
		case "bool":
			_, typeMatches = val.(bool)
		case "list":
			_, typeMatches = val.([]any)
		case "map":
			_, typeMatches = val.(map[string]any)
		}

		if !typeMatches {
			return fmt.Errorf("key '%s' has wrong type: expected %s, got %s", key, expectedType, actualType)
		}
	}
	return nil
}

func ApplyDefaults(filepath string, defaults map[string]any, policy DefaultPolicy) error {
	cfg, err := Load(filepath)
	if err != nil {
		return err
	}

	config := &Config{
		data:        cfg,
		defaultKeys: make(map[string]bool),
		mapStyle:    MapStyleAuto,
	}

	if policy.SkipIfPresent && len(config.data) > 0 {
		return nil
	}

	for key, value := range defaults {
		if err := config.applyDefault(key, value, policy); err != nil {
			return err
		}
	}

	return config.SaveToFile(filepath)
}

func (c *Config) applyDefault(key string, value any, policy DefaultPolicy) error {
	exists := c.Has(key)

	if policy.OnlyMissing && exists {
		return nil
	}

	if !policy.Override && exists {
		return nil
	}

	if policy.StrictTypes && exists {
		currentVal, _ := c.Get(key)
		if !typesMatch(currentVal, value) {
			return fmt.Errorf("type mismatch for key '%s': expected %T, got %T", key, currentVal, value)
		}
	}

	c.Set(key, value)
	c.defaultKeys[key] = true
	return nil
}

func typesMatch(a, b any) bool {
	return reflect.TypeOf(a) == reflect.TypeOf(b)
}

func (c *Config) SaveToFile(filepath string) error {
	content := c.Dump()
	return os.WriteFile(filepath, []byte(content), 0644)
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

func Cache(filepath string) (map[string]any, error) {
	cacheMutex.RLock()
	if cached, exists := configCache[filepath]; exists {
		cacheMutex.RUnlock()
		return cached, nil
	}
	cacheMutex.RUnlock()

	data, err := Load(filepath)
	if err != nil {
		return nil, err
	}

	cacheMutex.Lock()
	configCache[filepath] = data
	cacheMutex.Unlock()

	return data, nil
}

func Reload(filepath string) (map[string]any, error) {
	data, err := Load(filepath)
	if err != nil {
		return nil, err
	}

	cacheMutex.Lock()
	configCache[filepath] = data
	cacheMutex.Unlock()

	return data, nil
}

func ReloadKeys(filepath string, keys ...string) (map[string]any, error) {
	fresh, err := Load(filepath)
	if err != nil {
		return nil, err
	}

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if configCache[filepath] == nil {
		configCache[filepath] = make(map[string]any)
	}

	for _, key := range keys {
		if val, exists := fresh[key]; exists {
			configCache[filepath][key] = val
		}
	}

	return configCache[filepath], nil
}

func ClearCache() {
	cacheMutex.Lock()
	configCache = make(map[string]map[string]any)
	cacheMutex.Unlock()
}

func Load(filepath string) (map[string]any, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	cfg := New()
	if err := cfg.Parse(string(content)); err != nil {
		return nil, err
	}

	return cfg.data, nil
}

func (c *Config) ToJSON() (string, error) {
	jsonBytes, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func (c *Config) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), &c.data)
}
