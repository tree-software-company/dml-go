package dml

import (
	"fmt"
	"reflect"
)

func Unmarshal[T any](data []byte) (*T, error) {
	cfg, err := parseDocument(data)
	if err != nil {
		return nil, err
	}
	return Decode[T](cfg)
}

func UnmarshalFile[T any](path string) (*T, error) {
	cfg, err := loadLayer(path)
	if err != nil {
		return nil, fmt.Errorf("dml: %q: %w", path, err)
	}
	return Decode[T](cfg)
}

func Decode[T any](c *Config) (*T, error) {
	if c == nil {
		return nil, ErrNilConfig
	}

	result := new(T)
	if err := bindValue(reflect.ValueOf(result).Elem(), reflect.ValueOf(c.data), "", 0); err != nil {
		return nil, err
	}
	if err := validateDecoded(reflect.ValueOf(result).Elem(), reflect.ValueOf(c.data)); err != nil {
		return nil, err
	}
	return result, nil
}

func parseDocument(data []byte) (*Config, error) {
	if len(data) > maxLayerFileSize {
		return nil, ErrLayerTooLarge
	}

	cfg := New()
	if err := cfg.Parse(string(data)); err != nil {
		return nil, redactLayerError(err)
	}
	return cfg, nil
}
