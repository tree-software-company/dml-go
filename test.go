package main

import (
    "fmt"
    "log"

    "github.com/tree-software-company/dml-go/dml"
)

func loadConfig() *dml.Config {
    cfg, err := dml.NewConfig("testdata/example.dml")
    if err != nil {
        fmt.Println("❌ Error loading config:", err)
        return nil
    }
    fmt.Println("📄 Reloaded config! Keys:", cfg.Keys())
    return cfg
}

func main() {
    cfg := loadConfig()

    err := dml.Watch("testdata/example.dml", func() {
        newCfg := loadConfig()
        if newCfg != nil {
            cfg = newCfg
        }
    })
    if err != nil {
        log.Fatal(err)
    }

    if cfg != nil {
        fmt.Println("✅ Initial config loaded with keys:", cfg.Keys())
    }

    fmt.Println("👀 Watching testdata/example.dml. Press ENTER to exit.")
    fmt.Scanln()
}
