package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tree-software-company/dml-go/dml"
)

func main() {
	fmt.Println("=== DML ReloadKeys — Partial Hot-Reload Example ===")
	fmt.Println()

	tmpFile, err := os.CreateTemp("", "config-*.dml")
	if err != nil {
		log.Fatal(err)
	}
	configPath := tmpFile.Name()
	defer os.Remove(configPath)

	initialContent := `
map server = {"host": "localhost", "port": 8080};
map database = {"host": "db-primary", "port": 5432};
string app_version = "1.0.0";
`
	if _, err := tmpFile.WriteString(initialContent); err != nil {
		log.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := dml.NewConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println("✅ Initial config loaded:")
	printConfig(cfg)

	go func() {
		time.Sleep(200 * time.Millisecond)

		updatedContent := `
map server = {"host": "prod-server-01", "port": 9090};
map database = {"host": "db-replica", "port": 5433};
string app_version = "2.0.0";
`
		if err := os.WriteFile(configPath, []byte(updatedContent), 0o644); err != nil {
			log.Printf("Error writing updated config: %v", err)
		}
		fmt.Println("\n📝 Config file updated on disk (server + database + app_version changed)")
	}()

	fmt.Println("\n🔄 Starting partial reload loop (reloading 'server' and 'database' only)…")

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if err := cfg.ReloadKeys(configPath, "server", "database"); err != nil {
			log.Printf("ReloadKeys error: %v", err)
			continue
		}

		fmt.Println("\n✅ After partial reload:")
		printConfig(cfg)

		break
	}

	fmt.Println("\n--- Package-level ReloadKeys (cache-based) ---")

	if _, err := dml.Cache(configPath); err != nil {
		log.Fatalf("Cache: %v", err)
	}

	result, err := dml.ReloadKeys(configPath, "server")
	if err != nil {
		log.Fatalf("ReloadKeys: %v", err)
	}

	srv, _ := result["server"].(map[string]any)
	fmt.Printf("✅ server.host (cache) = %v\n", srv["host"])
	fmt.Println("\nDone.")
}

func printConfig(cfg *dml.Config) {
	srv := cfg.GetMap("server")
	db := cfg.GetMap("database")
	ver := cfg.GetString("app_version")

	fmt.Printf("  server.host     = %v\n", srv["host"])
	fmt.Printf("  server.port     = %v\n", srv["port"])
	fmt.Printf("  database.host   = %v\n", db["host"])
	fmt.Printf("  app_version     = %s  (not in reload list — stays unchanged)\n", ver)
}
