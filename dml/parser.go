package dml

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"

    "github.com/tree-software-company/dml-go/internal"
)

func Parse(filename string) (map[string]any, error) {
    jar, err := internal.EnsureJar()
    if err != nil {
        return nil, err
    }

    cmd := exec.Command("java", "-jar", jar, "-w", "json", filename)
    if out, err := cmd.CombinedOutput(); err != nil {
        return nil, fmt.Errorf("dml execution failed:\n%s", string(out))
    }

    jsonFile := filename[:len(filename)-len(filepath.Ext(filename))] + ".json"
    content, err := os.ReadFile(jsonFile)
    if err != nil {
        return nil, fmt.Errorf("failed to read generated json: %w", err)
    }

    var result map[string]any
    if err := json.Unmarshal(content, &result); err != nil {
        return nil, fmt.Errorf("invalid JSON: %w", err)
    }

    _ = os.Remove(jsonFile)

    return result, nil
}
