package dml

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidPath = errors.New("invalid file path")

func validatePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("%w: path is empty", ErrInvalidPath)
	}
	if strings.ContainsRune(path, 0) {
		return fmt.Errorf("%w: path contains a NUL byte", ErrInvalidPath)
	}
	for _, segment := range strings.FieldsFunc(path, isPathSeparator) {
		if segment == ".." {
			return fmt.Errorf("%w: parent directory references are not allowed", ErrInvalidPath)
		}
	}
	return nil
}

func isPathSeparator(r rune) bool {
	return r == '/' || r == '\\'
}
