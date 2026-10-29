package dml

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrValidation  = errors.New("value violates a validation rule")
	ErrInvalidRule = errors.New("invalid validation rule")
)

const maxReportedViolations = 10

type FieldError struct {
	Path  string
	Rule  string
	Param string
}

func (e *FieldError) Error() string {
	return "dml: validate " + e.describe()
}

func (e *FieldError) Unwrap() error {
	return ErrValidation
}

func (e *FieldError) describe() string {
	if e.Param == "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Rule)
	}
	return fmt.Sprintf("%s: %s=%s", e.Path, e.Rule, e.Param)
}

type ValidationError struct {
	Fields []*FieldError
}

func (e *ValidationError) Error() string {
	shown := e.Fields
	if len(shown) > maxReportedViolations {
		shown = shown[:maxReportedViolations]
	}

	descriptions := make([]string, len(shown))
	for i, field := range shown {
		descriptions[i] = field.describe()
	}

	message := fmt.Sprintf("dml: validation failed (%d): %s", len(e.Fields), strings.Join(descriptions, "; "))
	if hidden := len(e.Fields) - len(shown); hidden > 0 {
		message += fmt.Sprintf("; and %d more", hidden)
	}
	return message
}

func (e *ValidationError) Unwrap() []error {
	errs := make([]error, len(e.Fields))
	for i, field := range e.Fields {
		errs[i] = field
	}
	return errs
}

func newFieldError(path, rule, param string) *FieldError {
	return &FieldError{Path: displayPath(path), Rule: rule, Param: param}
}
