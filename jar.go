package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	jarURL    = "https://github.com/tree-software-company/DML/releases/download/0.4.1/DML-all.jar"
	jarSHA256 = "2a510ab80494be0a1e1431d222a6797407ce458a99726e814c57c6cd57f05919"
	jarName   = "DML-all.jar"
)

func ensureJar() (string, error) {
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

	fmt.Println("🔽 Downloading DML-all.jar...")
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
		return "", fmt.Errorf("SHA256 mismatch — file may be corrupted")
	}

	return jarPath, nil
}

func verifySHA256(path string, expected string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, err
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	return sum == expected, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: dml [command] [args...]")
		os.Exit(1)
	}

	jarPath, err := ensureJar()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to prepare DML JAR: %v\n", err)
		os.Exit(1)
	}

	args := append([]string{"-jar", jarPath}, os.Args[1:]...)
	cmd := exec.Command("java", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err = cmd.Run()
	if err != nil {
		os.Exit(1)
	}
}
