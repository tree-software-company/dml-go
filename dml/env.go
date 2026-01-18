package dml

import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

func LoadEnv(filepath string) error {
    file, err := os.Open(filepath)
    if err != nil {
        return fmt.Errorf("failed to open .env file: %w", err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    lineNum := 0

    for scanner.Scan() {
        lineNum++
        line := strings.TrimSpace(scanner.Text())

        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }

        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            return fmt.Errorf("invalid .env format at line %d: %s", lineNum, line)
        }

        key := strings.TrimSpace(parts[0])
        value := strings.TrimSpace(parts[1])

        value = strings.Trim(value, `"'`)

        if err := os.Setenv(key, value); err != nil {
            return fmt.Errorf("failed to set env var %s: %w", key, err)
        }
    }

    if err := scanner.Err(); err != nil {
        return fmt.Errorf("error reading .env file: %w", err)
    }

    return nil
}

func LoadEnvIfExists(filepath string) error {
    if _, err := os.Stat(filepath); os.IsNotExist(err) {
        return nil
    }
    return LoadEnv(filepath)
}

func ExpandEnv(s string) string {
    return os.ExpandEnv(s)
}

func GetEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func MustGetEnv(key string) string {
    value := os.Getenv(key)
    if value == "" {
        panic(fmt.Sprintf("required environment variable %s is not set", key))
    }
    return value
}

func (c *Config) LoadWithEnv() {
    c.expandValues(c.data)
}

func (c *Config) expandValues(data map[string]interface{}) {
    for key, value := range data {
        switch v := value.(type) {
        case string:
            data[key] = ExpandEnv(v)
        case map[string]interface{}:
            c.expandValues(v)
        case []interface{}:
            for i, item := range v {
                if str, ok := item.(string); ok {
                    v[i] = ExpandEnv(str)
                }
            }
        }
    }
}

func (c *Config) SetEnvDefaults(prefix string) error {
    return c.setEnvFromMap(c.data, prefix)
}

func (c *Config) setEnvFromMap(data map[string]interface{}, prefix string) error {
    for key, value := range data {
        envKey := key
        if prefix != "" {
            envKey = prefix + "_" + key
        }
        envKey = strings.ToUpper(envKey)

        switch v := value.(type) {
        case string:
            if os.Getenv(envKey) == "" {
                if err := os.Setenv(envKey, v); err != nil {
                    return err
                }
            }
        case int:
            if os.Getenv(envKey) == "" {
                if err := os.Setenv(envKey, fmt.Sprintf("%d", v)); err != nil {
                    return err
                }
            }
        case float64:
            if os.Getenv(envKey) == "" {
                if err := os.Setenv(envKey, fmt.Sprintf("%f", v)); err != nil {
                    return err
                }
            }
        case bool:
            if os.Getenv(envKey) == "" {
                if err := os.Setenv(envKey, fmt.Sprintf("%t", v)); err != nil {
                    return err
                }
            }
        case map[string]interface{}:
            if err := c.setEnvFromMap(v, envKey); err != nil {
                return err
            }
        }
    }
    return nil
}

func (c *Config) EnvOverride(prefix string) {
    c.envOverrideMap(c.data, prefix, "")
}

func (c *Config) envOverrideMap(data map[string]any, prefix string, parentKey string) {
    for key := range data {
        var envKey string
        if parentKey == "" {
            if prefix != "" {
                envKey = prefix + "_" + key
            } else {
                envKey = key
            }
        } else {
            envKey = parentKey + "_" + key
        }

        envKey = strings.ToUpper(envKey)

        if envValue := os.Getenv(envKey); envValue != "" {
            switch data[key].(type) {
            case int:
                var intVal int
                if _, err := fmt.Sscanf(envValue, "%d", &intVal); err == nil {
                    data[key] = intVal
                }
            case float64:
                var floatVal float64
                if _, err := fmt.Sscanf(envValue, "%f", &floatVal); err == nil {
                    data[key] = floatVal
                }
            case bool:
                switch strings.ToLower(envValue) {
                case "true", "1", "yes", "on":
                    data[key] = true
                case "false", "0", "no", "off":
                    data[key] = false
                }
            default:
                data[key] = envValue
            }
        }

        if nestedMap, ok := data[key].(map[string]any); ok {
            c.envOverrideMap(nestedMap, prefix, envKey)
        }
    }
}

func parseInt(value string, lineNum, col int, line string) (int, error) {
    var num int
    _, err := fmt.Sscanf(value, "%d", &num)
    return num, err
}

func parseFloat(value string, lineNum, col int, line string) (float64, error) {
    var num float64
    _, err := fmt.Sscanf(value, "%f", &num)
    return num, err
}

func parseBool(value string, lineNum, col int, line string) (bool, error) {
    switch strings.ToLower(value) {
    case "true", "1", "yes", "on":
        return true, nil
    case "false", "0", "no", "off":
        return false, nil
    default:
        return false, fmt.Errorf("invalid boolean value: %s", value)
    }
}