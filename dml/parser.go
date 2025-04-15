package dml

import (
    "encoding/json"
    "fmt"
    "os/exec"
    "github.com/tree-software-company/dml-go/internal"
)

func Parse(filename string) (map[string]any, error) {
    jar, err := internal.EnsureJar()
    if err != nil {
        return nil, err
    }

    cmd := exec.Command("java", "-jar", jar, "-w", "json", filename)
    output, err := cmd.CombinedOutput() 
    if err != nil {
        return nil, fmt.Errorf("dml execution failed:\n%s", string(output))
    }
    

    var result map[string]any
    if err := json.Unmarshal(output, &result); err != nil {
        return nil, err
    }
    return result, nil
}
