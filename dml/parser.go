package dml

import (
    "encoding/json"
    "os/exec"
    "github.com/tree-software-company/dml-go/internal"
)

func Parse(filename string) (map[string]any, error) {
    jar, err := internal.EnsureJar()
    if err != nil {
        return nil, err
    }

    cmd := exec.Command("java", "-jar", jar, "-w", "json", filename)
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    var result map[string]any
    if err := json.Unmarshal(output, &result); err != nil {
        return nil, err
    }
    return result, nil
}
