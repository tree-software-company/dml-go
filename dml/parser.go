package dml

import (
	"fmt"
	"strconv"
	"strings"
)

func (c *Config) Parse(content string) error {
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		line = strings.TrimSuffix(line, ";")

		if err := c.parseLine(line, lineNum+1, lines[lineNum]); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) parseLine(line string, lineNum int, originalLine string) error {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return newSyntaxError(
			lineNum,
			1,
			"Expected format: 'type name = value'",
			originalLine,
		)
	}

	declaration := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	declParts := strings.Fields(declaration)
	if len(declParts) != 2 {
		return newSyntaxError(
			lineNum,
			1,
			"Declaration must have format: 'type name'",
			originalLine,
		)
	}

	varType := declParts[0]
	varName := declParts[1]

	if !isValidIdentifier(varName) {
		col := strings.Index(originalLine, varName) + 1
		return newValidationError(
			lineNum,
			col,
			fmt.Sprintf("Invalid identifier: '%s'. Must start with letter and contain only letters, numbers, and underscores", varName),
			originalLine,
		)
	}

	parsedValue, err := c.parseValue(varType, value, lineNum, originalLine)
	if err != nil {
		return err
	}

	c.data[varName] = parsedValue
	return nil
}

func (c *Config) parseValue(varType, value string, lineNum int, originalLine string) (interface{}, error) {
	valueCol := strings.Index(originalLine, value) + 1

	switch varType {
	case "string":
		return c.parseString(value, lineNum, valueCol, originalLine)
	case "int":
		return c.parseInt(value, lineNum, valueCol, originalLine)
	case "float":
		return c.parseFloat(value, lineNum, valueCol, originalLine)
	case "bool":
		return c.parseBool(value, lineNum, valueCol, originalLine)
	case "list":
		return c.parseList(value, lineNum, valueCol, originalLine)
	case "map":
		return c.parseMap(value, lineNum, valueCol, originalLine)
	default:
		return nil, newValidationError(
			lineNum,
			1,
			fmt.Sprintf("Unknown type: '%s'. Valid types: string, int, float, bool, list, map", varType),
			originalLine,
		)
	}
}

func (c *Config) parseString(value string, lineNum, col int, line string) (string, error) {
	if !strings.HasPrefix(value, `"`) || !strings.HasSuffix(value, `"`) {
		return "", newTypeError(
			lineNum,
			col,
			"String must be enclosed in double quotes",
			line,
		)
	}
	return strings.Trim(value, `"`), nil
}

func (c *Config) parseInt(value string, lineNum, col int, line string) (int, error) {
	num, err := strconv.Atoi(value)
	if err != nil {
		return 0, newTypeError(
			lineNum,
			col,
			fmt.Sprintf("Invalid integer: '%s'", value),
			line,
		)
	}
	return num, nil
}

func (c *Config) parseFloat(value string, lineNum, col int, line string) (float64, error) {
	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, newTypeError(
			lineNum,
			col,
			fmt.Sprintf("Invalid float: '%s'", value),
			line,
		)
	}
	return num, nil
}

func (c *Config) parseBool(value string, lineNum, col int, line string) (bool, error) {
	switch value {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, newTypeError(
			lineNum,
			col,
			fmt.Sprintf("Invalid boolean: '%s'. Must be 'true' or 'false'", value),
			line,
		)
	}
}

func (c *Config) parseList(value string, lineNum, col int, line string) ([]interface{}, error) {
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil, newTypeError(
			lineNum,
			col,
			"List must be enclosed in square brackets []",
			line,
		)
	}

	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	value = strings.TrimSpace(value)

	if value == "" {
		return []interface{}{}, nil
	}

	items := strings.Split(value, ",")
	result := make([]interface{}, len(items))

	for i, item := range items {
		item = strings.TrimSpace(item)
		result[i] = strings.Trim(item, `"`)
	}

	return result, nil
}

func (c *Config) parseMap(value string, lineNum, col int, line string) (map[string]interface{}, error) {
	if !strings.HasPrefix(value, "{") || !strings.HasSuffix(value, "}") {
		return nil, newTypeError(
			lineNum,
			col,
			"Map must be enclosed in curly braces {}",
			line,
		)
	}

	value = strings.TrimPrefix(value, "{")
	value = strings.TrimSuffix(value, "}")
	value = strings.TrimSpace(value)

	if value == "" {
		return map[string]interface{}{}, nil
	}

	result := make(map[string]interface{})
	pairs := strings.Split(value, ",")

	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) != 2 {
			return nil, newSyntaxError(
				lineNum,
				col,
				"Map entries must have format: \"key\": \"value\"",
				line,
			)
		}

		key := strings.TrimSpace(kv[0])
		key = strings.Trim(key, `"'`)

		val := strings.TrimSpace(kv[1])
		val = strings.Trim(val, `"'`)

		result[key] = val
	}

	return result, nil
}

func isValidIdentifier(name string) bool {
	if name == "" {
		return false
	}

	first := rune(name[0])
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}

	for _, ch := range name[1:] {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}

	return true
}
