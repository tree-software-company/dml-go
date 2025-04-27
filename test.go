package main

import (
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/tree-software-company/dml-go/dml"
)

func main() {
    cfg, err := dml.NewConfig("testdata/config.dml")
    if err != nil {
        log.Fatal("❌ Failed to load config:", err)
    }

    if err := cfg.ValidateRequiredTyped(map[string]string{
        "server": "map[string]interface {}",
    }); err != nil {
        log.Fatal(err)
    }

    serverMap := cfg.GetMap("server")
    if serverMap == nil {
        log.Fatal("❌ 'server' config missing!")
    }

    if _, ok := serverMap["port"]; !ok {
        log.Fatal("❌ 'server.port' missing!")
    }
    if _, ok := serverMap["timeout"]; !ok {
        log.Fatal("❌ 'server.timeout' missing!")
    }

    port := int(serverMap["port"].(float64))
    timeoutSeconds := int(serverMap["timeout"].(float64))

    fmt.Printf("🚀 Starting server on port %d\n", port)
    fmt.Printf("⏳ Timeout: %ds\n", timeoutSeconds)

    server := &http.Server{
        Addr:         fmt.Sprintf(":%d", port),
        ReadTimeout:  time.Duration(timeoutSeconds) * time.Second,
        WriteTimeout: time.Duration(timeoutSeconds) * time.Second,
    }

    http.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "👋 Hello from DML-configured server!")
    })

    if err := server.ListenAndServe(); err != nil {
        log.Fatal("❌ Server error:", err)
    }
}
