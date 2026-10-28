package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tree-software-company/dml-go/dml"
)

func main() {
	fmt.Println("=== DML Watch — Live Reload Example ===")
	fmt.Println()

	tmpFile, err := os.CreateTemp("", "config-*.dml")
	if err != nil {
		log.Fatal(err)
	}
	configPath := tmpFile.Name()
	defer os.Remove(configPath)

	initialContent := `
map server = {"host": "localhost", "port": 8080};
string app_version = "1.0.0";
`
	if _, err := tmpFile.WriteString(initialContent); err != nil {
		log.Fatal(err)
	}
	tmpFile.Close()

	reloaded := make(chan struct{}, 1)

	watcher, err := dml.Watch(configPath, dml.WatchOptions{
		Interval: 100 * time.Millisecond,
		OnReload: func(cfg *dml.Config) {
			fmt.Println("\n🔄 Reload detected, new snapshot installed")
			select {
			case reloaded <- struct{}{}:
			default:
			}
		},
		OnError: func(err error) {
			log.Printf("watch error: %v", err)
		},
	})
	if err != nil {
		log.Fatalf("Failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	fmt.Println("✅ Initial config loaded:")
	printWatched(watcher)

	go func() {
		time.Sleep(300 * time.Millisecond)

		updatedContent := `
map server = {"host": "prod-server-01", "port": 9090};
string app_version = "2.0.0";
`
		if err := os.WriteFile(configPath, []byte(updatedContent), 0o644); err != nil {
			log.Printf("Error writing updated config: %v", err)
			return
		}
		fmt.Println("\n📝 Config file updated on disk")
	}()

	select {
	case <-reloaded:
		fmt.Println("\n✅ After live reload (no restart):")
		printWatched(watcher)
	case <-time.After(5 * time.Second):
		log.Fatal("timed out waiting for the reload")
	}

	fmt.Println("\nDone.")
}

func printWatched(watcher *dml.Watcher) {
	srv := watcher.GetMap("server")

	fmt.Printf("  server.host = %v\n", srv["host"])
	fmt.Printf("  server.port = %v\n", srv["port"])
	fmt.Printf("  app_version = %s\n", watcher.GetString("app_version"))
}
