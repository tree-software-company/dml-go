package dml

import (
	"fmt"
	"strconv"
	"strings"
)

func (c *Config) Parse(content string) error {
	lines := strings.Split(content, "\n")
	var multiLineBuffer strings.Builder
	var isInMultiLine bool
	var multiLineStart int

	for lineNum, line := range lines {
		originalLine := line
		line = strings.TrimSpace(line)

		// Skip empty lines and comments when not in multi-line
		if !isInMultiLine && (line == "" || strings.HasPrefix(line, "//")) {
			continue
		}

		// Handle directives (only at top level)
		if !isInMultiLine && strings.HasPrefix(line, "@") {
			if err := c.parseDirective(line, lineNum+1); err != nil {
				return err
			}
			continue
		}

		// Check if we're starting a multi-line declaration
		if !isInMultiLine && (strings.Contains(line, "{") || strings.Contains(line, "[")) {
			if !strings.HasSuffix(line, ";") {
				isInMultiLine = true
				multiLineStart = lineNum + 1
				multiLineBuffer.WriteString(line)
				multiLineBuffer.WriteString(" ")
				continue
			}
		}

		// Continue collecting multi-line content
		if isInMultiLine {
			multiLineBuffer.WriteString(line)
			multiLineBuffer.WriteString(" ")

			// Check if multi-line ends
			if strings.HasSuffix(line, ";") {
				fullLine := multiLineBuffer.String()
				if err := c.parseLine(fullLine, multiLineStart, fullLine); err != nil {
					return err
				}
				multiLineBuffer.Reset()
				isInMultiLine = false
			}
			continue
		}

		// Single-line declaration
		if !strings.HasSuffix(line, ";") {
			return &DMLError{
				Type:    ErrorTypeSyntax,
				Message: "Missing semicolon at the end of declaration",
				Line:    lineNum + 1,
				Column:  len(line),
				Context: originalLine,
			}
		}

		if err := c.parseLine(line, lineNum+1, originalLine); err != nil {
			return err
		}
	}

	// Check for unclosed multi-line
	if isInMultiLine {
		return &DMLError{
			Type:    ErrorTypeSyntax,
			Message: "Unclosed declaration (missing ';')",
			Line:    multiLineStart,
			Column:  1,
			Context: multiLineBuffer.String(),
		}
	}

	return nil
}

func (c *Config) parseDirective(line string, lineNum int) error {
	line = strings.TrimSpace(strings.TrimPrefix(line, "@"))
	parts := strings.Fields(line)

	if len(parts) < 2 {
		return &DMLError{
			Type:    ErrorTypeValidation,
			Message: fmt.Sprintf("Invalid directive: @%s", line),
			Line:    lineNum,
			Column:  1,
			Context: line,
		}
	}

	directive := parts[0]
	value := parts[1]

	switch directive {
	case "mapStyle":
		return c.handleMapStyleDirective(value, lineNum)
	default:
		return &DMLError{
			Type:    ErrorTypeValidation,
			Message: fmt.Sprintf("Unknown directive: @%s", directive),
			Line:    lineNum,
			Column:  1,
			Context: line,
		}
	}
}

func (c *Config) handleMapStyleDirective(value string, lineNum int) error {
	switch strings.ToLower(value) {
	case "json":
		c.SetMapStyle(MapStyleJSON)
	case "flat":
		c.SetMapStyle(MapStyleFlat)
	case "auto":
		c.SetMapStyle(MapStyleAuto)
	default:
		return &DMLError{
			Type:    ErrorTypeValidation,
			Message: fmt.Sprintf("Invalid mapStyle value: %s (expected: json, flat, or auto)", value),
			Line:    lineNum,
			Column:  1,
			Context: value,
		}
	}
	return nil
}

func (c *Config) parseLine(line string, lineNum int, originalLine string) error {
	line = strings.TrimSuffix(strings.TrimSpace(line), ";")

	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return &DMLError{
			Type:    ErrorTypeSyntax,
			Message: "Missing '=' operator in variable declaration",
			Line:    lineNum,
			Column:  1,
			Context: originalLine,
		}
	}

	declaration := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	declParts := strings.Fields(declaration)
	if len(declParts) != 2 {
		return &DMLError{
			Type:    ErrorTypeValidation,
			Message: "Invalid variable declaration format. Expected: type name = value",
			Line:    lineNum,
			Column:  1,
			Context: originalLine,
		}
	}

	varType := declParts[0]
	varName := declParts[1]

	if !isValidIdentifier(varName) {
		col := strings.Index(originalLine, varName) + 1
		if col == 0 {
			col = 1
		}
		return &DMLError{
			Type:    ErrorTypeValidation,
			Message: "Invalid identifier. Must start with letter or underscore, and contain only letters, digits, underscores, or dots",
			Line:    lineNum,
			Column:  col,
			Context: originalLine,
		}
	}

	parsedValue, err := c.parseValue(varType, value, lineNum, 1, originalLine)
	if err != nil {
		return err
	}

	c.Set(varName, parsedValue)
	return nil
}

func (c *Config) parseValue(varType, value string, lineNum, col int, line string) (interface{}, error) {
	switch varType {
	case "string":
		return c.parseString(value, lineNum, col, line)
	case "int", "number":
		return c.parseInt(value, lineNum, col, line)
	case "float":
		return c.parseFloat(value, lineNum, col, line)
	case "bool", "boolean":
		return c.parseBool(value, lineNum, col, line)
	case "list":
		return c.parseList(value, lineNum, col, line)
	case "map":
		return c.parseMap(value, lineNum, col, line)
	default:
		return nil, &DMLError{
			Type:    ErrorTypeValidation,
			Message: fmt.Sprintf("Unknown type: %s", varType),
			Line:    lineNum,
			Column:  col,
			Context: line,
		}
	}
}

func (c *Config) parseString(value string, lineNum, col int, line string) (string, error) {
	if !strings.HasPrefix(value, `"`) || !strings.HasSuffix(value, `"`) {
		return "", &DMLError{
			Type:    ErrorTypeType,
			Message: "String must be enclosed in double quotes",
			Line:    lineNum,
			Column:  col,
			Context: line,
		}
	}
	return strings.Trim(value, `"`), nil
}

func (c *Config) parseInt(value string, lineNum, col int, line string) (int, error) {
	val, err := strconv.Atoi(value)
	if err != nil {
		return 0, &DMLError{
			Type:    ErrorTypeType,
			Message: fmt.Sprintf("Invalid integer value: %s", value),
			Line:    lineNum,
			Column:  col,
			Context: line,
		}
	}
	return val, nil
}

func (c *Config) parseFloat(value string, lineNum, col int, line string) (float64, error) {
	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, &DMLError{
			Type:    ErrorTypeType,
			Message: fmt.Sprintf("Invalid float value: %s", value),
			Line:    lineNum,
			Column:  col,
			Context: line,
		}
	}
	return val, nil
}

func (c *Config) parseBool(value string, lineNum, col int, line string) (bool, error) {
	switch value {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, &DMLError{
			Type:    ErrorTypeType,
			Message: "Boolean must be 'true' or 'false'",
			Line:    lineNum,
			Column:  col,
			Context: line,
		}
	}
}

func (c *Config) parseList(value string, lineNum, col int, line string) ([]interface{}, error) {
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil, &DMLError{
			Type:    ErrorTypeType,
			Message: "List must be enclosed in square brackets []",
			Line:    lineNum,
			Column:  col,
			Context: line,
		}
	}

	content := strings.Trim(value, "[]")
	content = strings.TrimSpace(content)

	if content == "" {
		return []interface{}{}, nil
	}

	items := strings.Split(content, ",")
	result := make([]interface{}, 0, len(items))

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		if strings.HasPrefix(item, `"`) && strings.HasSuffix(item, `"`) {
			result = append(result, strings.Trim(item, `"`))
		} else if val, err := strconv.Atoi(item); err == nil {
			result = append(result, val)
		} else if val, err := strconv.ParseFloat(item, 64); err == nil {
			result = append(result, val)
		} else if item == "true" || item == "false" {
			result = append(result, item == "true")
		} else {
			result = append(result, item)
		}
	}

	return result, nil
}

func (c *Config) parseMap(value string, lineNum, col int, line string) (map[string]interface{}, error) {
	if !strings.HasPrefix(value, "{") || !strings.HasSuffix(value, "}") {
		return nil, &DMLError{
			Type:    ErrorTypeType,
			Message: "Map must be enclosed in curly braces {}",
			Line:    lineNum,
			Column:  col,
			Context: line,
		}
	}

	content := strings.Trim(value, "{}")
	content = strings.TrimSpace(content)

	if content == "" {
		return map[string]interface{}{}, nil
	}

	result := make(map[string]interface{})
	pairs := c.smartSplit(content, ',')

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			return nil, &DMLError{
				Type:    ErrorTypeType,
				Message: "Map entries must be in 'key: value' format",
				Line:    lineNum,
				Column:  col,
				Context: line,
			}
		}

		key := strings.Trim(strings.TrimSpace(parts[0]), `"`)
		val := strings.TrimSpace(parts[1])

		parsedVal := c.parseMapValue(val)
		result[key] = parsedVal
	}

	return result, nil
}

func (c *Config) smartSplit(s string, delimiter rune) []string {
	var result []string
	var current strings.Builder
	depth := 0
	inQuotes := false

	for _, ch := range s {
		if ch == '"' {
			inQuotes = !inQuotes
			current.WriteRune(ch)
			continue
		}

		if !inQuotes {
			if ch == '{' || ch == '[' {
				depth++
			} else if ch == '}' || ch == ']' {
				depth--
			}
		}

		if ch == delimiter && depth == 0 && !inQuotes {
			result = append(result, current.String())
			current.Reset()
		} else {
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

func (c *Config) parseMapValue(val string) interface{} {
	val = strings.TrimSpace(val)

	if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) {
		return strings.Trim(val, `"`)
	}

	if val == "true" {
		return true
	}
	if val == "false" {
		return false
	}

	if intVal, err := strconv.Atoi(val); err == nil {
		return intVal
	}

	if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
		return floatVal
	}

	if strings.HasPrefix(val, "{") && strings.HasSuffix(val, "}") {
		if nested, err := c.parseMap(val, 0, 0, ""); err == nil {
			return nested
		}
	}

	if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
		if nested, err := c.parseList(val, 0, 0, ""); err == nil {
			return nested
		}
	}

	return val
}

func isValidIdentifier(name string) bool {
	if len(name) == 0 {
		return false
	}

	for i, ch := range name {
		if i == 0 {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_') {
				return false
			}
		} else {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '.') {
				return false
			}
		}
	}

	return true
}
