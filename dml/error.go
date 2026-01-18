package dml

import (
    "fmt"
    "strings"
)

type DMLError struct {
    Line    int
    Column  int
    Message string
    Context string
    Type    ErrorType
}

type ErrorType int

const (
    ErrorTypeSyntax ErrorType = iota
    ErrorTypeValidation
    ErrorTypeType
    ErrorTypeUnknown
)

func (t ErrorType) String() string {
    switch t {
    case ErrorTypeSyntax:
        return "Syntax Error"
    case ErrorTypeValidation:
        return "Validation Error"
    case ErrorTypeType:
        return "Type Error"
    default:
        return "Unknown Error"
    }
}

func (e *DMLError) Error() string {
    var sb strings.Builder
    
    sb.WriteString(fmt.Sprintf("%s at line %d:%d\n", e.Type, e.Line, e.Column))
    sb.WriteString(fmt.Sprintf("  %s\n", e.Message))
    
    if e.Context != "" {
        sb.WriteString(fmt.Sprintf("\n  %s\n", e.Context))
        sb.WriteString(fmt.Sprintf("  %s^\n", strings.Repeat(" ", e.Column-1)))
    }
    
    return sb.String()
}

func newSyntaxError(line, column int, message, context string) *DMLError {
    return &DMLError{
        Line:    line,
        Column:  column,
        Message: message,
        Context: context,
        Type:    ErrorTypeSyntax,
    }
}

func newValidationError(line, column int, message, context string) *DMLError {
    return &DMLError{
        Line:    line,
        Column:  column,
        Message: message,
        Context: context,
        Type:    ErrorTypeValidation,
    }
}

func newTypeError(line, column int, message, context string) *DMLError {
    return &DMLError{
        Line:    line,
        Column:  column,
        Message: message,
        Context: context,
        Type:    ErrorTypeType,
    }
}