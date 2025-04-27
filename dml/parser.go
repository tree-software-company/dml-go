package dml

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "reflect"

    "github.com/tree-software-company/dml-go/internal"
)

var cache = make(map[string]map[string]any)

func Parse(filename string) (map[string]any, error) {
    jar, err := internal.EnsureJar()
    if err != nil {
        return nil, fmt.Errorf("❌ Error preparing DML interpreter:\n↳ %w", err)
    }

    cmd := exec.Command("java", "-jar", jar, "-w", "json", filename)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("❌ Error parsing file %s\n↳ %s", filename, string(output))
    }

    jsonFile := filename[:len(filename)-len(filepath.Ext(filename))] + ".json"

    content, err := os.ReadFile(jsonFile)
    if err != nil {
        return nil, fmt.Errorf("❌ Error reading generated JSON file:\n↳ %w", err)
    }

    var result map[string]any
    if err := json.Unmarshal(content, &result); err != nil {
        return nil, fmt.Errorf("❌ Error parsing generated JSON for file %s\n↳ %w", filename, err)
    }

    _ = os.Remove(jsonFile) 

    return result, nil
}

func ParseInto(filename string, v any) error {
    data, err := Parse(filename)
    if err != nil {
        return err
    }

    jsonBytes, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("❌ Error serializing intermediate data:\n↳ %w", err)
    }

    if err := json.Unmarshal(jsonBytes, v); err != nil {
        return fmt.Errorf("❌ Error mapping DML data to Go struct:\n↳ %w", err)
    }

    return nil
}

func Cache(filename string) (map[string]any, error) {
    if data, ok := cache[filename]; ok {
        return data, nil 
    }

    parsed, err := Parse(filename)
    if err != nil {
        return nil, err
    }

    cache[filename] = parsed 
    return parsed, nil
}

func ClearCache() {
    cache = make(map[string]map[string]any)
}

func Reload(filename string) (map[string]any, error) {
    parsed, err := Parse(filename)
    if err != nil {
        return nil, err
    }

    cache[filename] = parsed
    return parsed, nil
}

var requiredSchema = map[string]string{
    "port":      "float64", 
    "debug":     "bool",
    "timeout":   "float64",
    "apiPrefix": "string",
}

func Load[T any](filename string) (T, error) {
    var result T

    data, err := Cache(filename)
    if err != nil {
        return result, err
    }

    for key, expectedType := range requiredSchema {
        value, ok := data[key]
        if !ok {
            return result, fmt.Errorf("❌ Missing required key '%s' in file '%s'", key, filename)
        }

        actualType := reflect.TypeOf(value).String()
        if actualType != expectedType {
            return result, fmt.Errorf("❌ Key '%s' must be of type '%s', but got '%s'", key, expectedType, actualType)
        }
    }

    bytes, err := json.Marshal(data)
    if err != nil {
        return result, err
    }

    if err := json.Unmarshal(bytes, &result); err != nil {
        return result, err
    }

    return result, nil
}

