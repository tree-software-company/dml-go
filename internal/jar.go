package internal

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
)

const (
    jarURL    = "https://github.com/tree-software-company/DML/releases/download/0.8.1/DML-all.jar"
    jarSHA256 = "ce6ed0f4fbb277937812fef53a82526a55f1b0dcd2b4f4692c92cd81a45bd9fd"
    jarName   = "DML-all.jar"
)

func EnsureJar() (string, error) {
    cacheDir, err := os.UserCacheDir()
    if err != nil {
        return "", err
    }
    dmlDir := filepath.Join(cacheDir, "dml-go")
    jarPath := filepath.Join(dmlDir, jarName)

    os.MkdirAll(dmlDir, 0755)

    if _, err := os.Stat(jarPath); err == nil {
        if valid, _ := verifySHA256(jarPath, jarSHA256); valid {
            return jarPath, nil
        }
    }

    fmt.Println("📦 Downloading DML interpreter...")
    out, err := os.Create(jarPath)
    if err != nil {
        return "", err
    }
    defer out.Close()

    resp, err := http.Get(jarURL)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    _, err = io.Copy(out, resp.Body)
    if err != nil {
        return "", err
    }

    if valid, _ := verifySHA256(jarPath, jarSHA256); !valid {
        return "", fmt.Errorf("SHA256 mismatch")
    }

    return jarPath, nil
}

func verifySHA256(path string, expected string) (bool, error) {
    f, err := os.Open(path)
    if err != nil {
        return false, err
    }
    defer f.Close()

    hash := sha256.New()
    if _, err := io.Copy(hash, f); err != nil {
        return false, err
    }
    sum := hex.EncodeToString(hash.Sum(nil))
    return sum == expected, nil
}
