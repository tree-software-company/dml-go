package main

import (
    "fmt"
    "log"
    "os"

    "github.com/tree-software-company/dml-go/dml"
)

func main() {
    fmt.Println("=== DML Environment Variables Integration ===\n")

    fmt.Println("📁 Example 1: Loading .env file")
    if err := dml.LoadEnvIfExists("testdata/.env.example"); err != nil {
        log.Printf("Warning: Could not load .env: %v\n", err)
    } else {
        fmt.Println("✅ .env file loaded successfully")
        fmt.Printf("  DB_HOST = %s\n", os.Getenv("DB_HOST"))
        fmt.Printf("  APP_NAME = %s\n", os.Getenv("APP_NAME"))
    }

    fmt.Println("\n📝 Example 2: Loading DML config with env interpolation")
    cfg, err := dml.NewConfig("testdata/config-with-env.dml")
    if err != nil {
        log.Fatal(err)
    }

    cfg.LoadWithEnv()

    fmt.Println("✅ Config loaded and env vars expanded:")
    fmt.Printf("  db_host = %s\n", cfg.GetString("db_host"))
    fmt.Printf("  db_url = %s\n", cfg.GetString("db_url"))
    fmt.Printf("  app_name = %s\n", cfg.GetString("app_name"))

    fmt.Println("\n🔧 Example 3: Using GetEnv with defaults")
    timeout := dml.GetEnv("REQUEST_TIMEOUT", "30")
    maxConns := dml.GetEnv("MAX_CONNECTIONS", "100")
    fmt.Printf("  REQUEST_TIMEOUT = %s (default)\n", timeout)
    fmt.Printf("  MAX_CONNECTIONS = %s (default)\n", maxConns)

    fmt.Println("\n⚙️  Example 4: Setting env defaults from DML config")
    cfg2 := dml.New()
    cfg2.Parse(`
        string server_host = "0.0.0.0";
        int server_port = 3000;
        bool server_tls = false;
    `)

    if err := cfg2.SetEnvDefaults("SERVER"); err != nil {
        log.Fatal(err)
    }

    fmt.Println("✅ Environment defaults set:")
    fmt.Printf("  SERVER_SERVER_HOST = %s\n", os.Getenv("SERVER_SERVER_HOST"))
    fmt.Printf("  SERVER_SERVER_PORT = %s\n", os.Getenv("SERVER_SERVER_PORT"))
    fmt.Printf("  SERVER_SERVER_TLS = %s\n", os.Getenv("SERVER_SERVER_TLS"))

    fmt.Println("\n🔄 Example 5: Overriding config from environment")
    os.Setenv("APP_PORT", "9000")
    os.Setenv("APP_DEBUG", "false")

    cfg3 := dml.New()
    cfg3.Parse(`
        string name = "MyApp";
        int port = 8080;
        bool debug = true;
    `)

    fmt.Println("Before override:")
    fmt.Printf("  port = %d\n", cfg3.GetInt("port"))
    fmt.Printf("  debug = %v\n", cfg3.GetBool("debug"))

    cfg3.EnvOverride("APP")

    fmt.Println("After override:")
    fmt.Printf("  port = %d\n", cfg3.GetInt("port"))
    fmt.Printf("  debug = %v\n", cfg3.GetBool("debug"))

    fmt.Println("\n🚀 Example 6: Full integration (12-factor app)")

    dml.LoadEnvIfExists(".env")

    appCfg, err := dml.NewConfig("testdata/config-with-env.dml")
    if err != nil {
        log.Fatal(err)
    }

    appCfg.LoadWithEnv()
    
    appCfg.EnvOverride("APP")
    
    fmt.Println("✅ Final configuration:")
    fmt.Printf("  Environment: %s\n", appCfg.GetString("app_env"))
    fmt.Printf("  Database: %s\n", appCfg.GetString("db_host"))
    fmt.Printf("  Port: %d\n", appCfg.GetInt("app_port"))
    fmt.Printf("  Debug: %v\n", appCfg.GetBool("app_debug"))

    fmt.Println("\n✨ All examples completed!")
}