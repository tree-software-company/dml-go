package dml

import (
    "strings"
    "testing"
)

func TestDMLError_Error(t *testing.T) {
    err := &DMLError{
        Line:    5,
        Column:  10,
        Message: "Invalid syntax",
        Context: "string name = invalid",
        Type:    ErrorTypeSyntax,
    }
    
    result := err.Error()
    
    if !strings.Contains(result, "line 5:10") {
        t.Errorf("Error message should contain line and column")
    }
    if !strings.Contains(result, "Invalid syntax") {
        t.Errorf("Error message should contain the message")
    }
    if !strings.Contains(result, "^") {
        t.Errorf("Error message should show position indicator")
    }
}

func TestErrorType_String(t *testing.T) {
    tests := []struct {
        errorType ErrorType
        expected  string
    }{
        {ErrorTypeSyntax, "Syntax Error"},
        {ErrorTypeValidation, "Validation Error"},
        {ErrorTypeType, "Type Error"},
        {ErrorTypeUnknown, "Unknown Error"},
    }
    
    for _, tt := range tests {
        if got := tt.errorType.String(); got != tt.expected {
            t.Errorf("ErrorType.String() = %v, want %v", got, tt.expected)
        }
    }
}