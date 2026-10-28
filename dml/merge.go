package dml

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
)

const (
	maxLayerFileSize     = 10 << 20
	layerRedactedMessage = "invalid content, see the reported position in the file"
)

var (
	ErrNoLayers          = errors.New("no files to load")
	ErrLayerTooLarge     = errors.New("file exceeds the maximum size")
	ErrInvalidAssignment = errors.New("invalid assignment, expected key=value")
	ErrInvalidOverride   = errors.New("value does not match the type of the existing key")
)

func LoadFiles(paths ...string) (*Config, error) {
	if len(paths) == 0 {
		return nil, ErrNoLayers
	}

	merged := New()
	for i, path := range paths {
		layer, err := loadLayer(path)
		if err != nil {
			return nil, fmt.Errorf("dml: file %d (%s): %w", i+1, path, err)
		}
		merged.Merge(layer)
	}

	return merged, nil
}

func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	mergeMaps(c.data, other.data)

	if other.mapStyle != MapStyleAuto {
		c.mapStyle = other.mapStyle
	}
}

func (c *Config) SetFromString(assignment string) error {
	key, raw, found := strings.Cut(assignment, "=")
	key = strings.TrimSpace(key)
	if !found || !isValidKeyPath(key) {
		return ErrInvalidAssignment
	}

	value, err := c.coerceOverride(key, raw)
	if err != nil {
		return err
	}

	c.Set(key, value)
	return nil
}

func loadLayer(path string) (*Config, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxLayerFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxLayerFileSize {
		return nil, ErrLayerTooLarge
	}

	cfg := New()
	if err := cfg.Parse(string(content)); err != nil {
		return nil, redactLayerError(err)
	}

	return cfg, nil
}

func redactLayerError(err error) error {
	var dmlErr *DMLError
	if errors.As(err, &dmlErr) {
		return &DMLError{
			Line:    dmlErr.Line,
			Column:  dmlErr.Column,
			Message: layerRedactedMessage,
			Type:    dmlErr.Type,
		}
	}
	return err
}

func mergeMaps(base, overlay map[string]any) {
	for key, value := range overlay {
		overlayMap, overlayIsMap := value.(map[string]any)
		baseMap, baseIsMap := base[key].(map[string]any)
		if overlayIsMap && baseIsMap {
			mergeMaps(baseMap, overlayMap)
			continue
		}
		base[key] = cloneValue(value)
	}
}

func cloneValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		copied := make(map[string]any, len(v))
		for key, item := range v {
			copied[key] = cloneValue(item)
		}
		return copied
	case []any:
		copied := make([]any, len(v))
		for i, item := range v {
			copied[i] = cloneValue(item)
		}
		return copied
	default:
		return value
	}
}

func isValidKeyPath(key string) bool {
	if !isValidIdentifier(key) {
		return false
	}
	for _, segment := range strings.Split(key, ".") {
		if segment == "" {
			return false
		}
	}
	return true
}

func (c *Config) coerceOverride(key, raw string) (any, error) {
	existing, exists := c.Get(key)
	if !exists {
		return inferScalar(raw), nil
	}

	switch existing.(type) {
	case int:
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return nil, overrideTypeError(key, "an integer")
		}
		return parsed, nil
	case float64:
		parsed, ok := parseFiniteFloat(raw)
		if !ok {
			return nil, overrideTypeError(key, "a number")
		}
		return parsed, nil
	case bool:
		parsed, ok := parseBoolWord(raw)
		if !ok {
			return nil, overrideTypeError(key, "a boolean")
		}
		return parsed, nil
	case map[string]any, []any:
		return nil, overrideTypeError(key, "a scalar value")
	default:
		return raw, nil
	}
}

func inferScalar(raw string) any {
	if parsed, err := strconv.Atoi(raw); err == nil {
		return parsed
	}
	if parsed, ok := parseFiniteFloat(raw); ok {
		return parsed
	}
	switch strings.ToLower(raw) {
	case "true":
		return true
	case "false":
		return false
	}
	return raw
}

func parseFiniteFloat(raw string) (float64, bool) {
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, false
	}
	return parsed, true
}

func parseBoolWord(raw string) (bool, bool) {
	switch strings.ToLower(raw) {
	case "true", "1", "yes", "on":
		return true, true
	case "false", "0", "no", "off":
		return false, true
	}
	return false, false
}

func overrideTypeError(key, expected string) error {
	return fmt.Errorf("%w: key %q expects %s", ErrInvalidOverride, key, expected)
}
