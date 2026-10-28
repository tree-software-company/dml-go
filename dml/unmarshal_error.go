package dml

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrTypeMismatch    = errors.New("value does not match the field type")
	ErrOutOfRange      = errors.New("number is out of range for the field type")
	ErrUnsupportedType = errors.New("unsupported field type")
	ErrMissingRequired = errors.New("required key is missing")
	ErrTooDeep         = errors.New("value is nested too deeply")
	ErrNilConfig       = errors.New("config is nil")
)

const rootBindPath = "root"

type BindError struct {
	Path   string
	Detail string
	Err    error
}

func (e *BindError) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("dml: bind %s: %v", e.Path, e.Err)
	}
	return fmt.Sprintf("dml: bind %s: %v (%s)", e.Path, e.Err, e.Detail)
}

func (e *BindError) Unwrap() error {
	return e.Err
}

func newBindError(path string, err error, detail string) *BindError {
	switch {
	case path == "":
		path = rootBindPath
	case strings.HasPrefix(path, "["):
		path = rootBindPath + path
	}
	return &BindError{Path: path, Detail: detail, Err: err}
}
