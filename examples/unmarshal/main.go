package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/tree-software-company/dml-go/dml"
)

type Database struct {
	Host string `dml:"host"`
	Port int    `dml:"port"`
}

type Server struct {
	Name    string   `dml:"name"`
	Port    int      `dml:"port"`
	Debug   bool     `dml:"debug"`
	Ratio   float64  `dml:"ratio"`
	Aliases []string `dml:"aliases"`
	DB      Database `dml:"db"`
}

type AppConfig struct {
	Name   string `dml:"name,required"`
	Server Server `dml:"server"`
}

const source = `
string name = "BeautyApp";
string server.name = "api";
int server.port = 8080;
bool server.debug = true;
float server.ratio = 0.75;
list server.aliases = ["api", "gateway"];
string server.db.host = "localhost";
int server.db.port = 5432;
`

const brokenSource = `
string name = "BeautyApp";
string server.port = "not a number";
`

func main() {
	fmt.Println("=== DML Typed Unmarshal ===")
	fmt.Println()

	cfg, err := dml.Unmarshal[AppConfig]([]byte(source))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("1. Unmarshal into a typed struct:")
	fmt.Printf("   name=%s server=%s port=%d debug=%t ratio=%.2f\n",
		cfg.Name, cfg.Server.Name, cfg.Server.Port, cfg.Server.Debug, cfg.Server.Ratio)
	fmt.Printf("   aliases=%v db=%s:%d\n", cfg.Server.Aliases, cfg.Server.DB.Host, cfg.Server.DB.Port)
	fmt.Println()

	fmt.Println("2. Decode a layered config:")
	layered, err := dml.LoadFiles("testdata/config.base.dml", "testdata/config.prod.dml")
	if err != nil {
		log.Fatal(err)
	}
	type Layered struct {
		Debug  bool   `dml:"debug"`
		Region string `dml:"region"`
		Server struct {
			Port int  `dml:"port"`
			TLS  bool `dml:"tls"`
		} `dml:"server"`
	}
	merged, err := dml.Decode[Layered](layered)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   debug=%t region=%s port=%d tls=%t\n", merged.Debug, merged.Region, merged.Server.Port, merged.Server.TLS)
	fmt.Println()

	fmt.Println("3. Type mismatch reports the field path:")
	_, err = dml.Unmarshal[AppConfig]([]byte(brokenSource))
	var bindErr *dml.BindError
	if errors.As(err, &bindErr) {
		fmt.Printf("   path=%s mismatch=%t\n", bindErr.Path, errors.Is(err, dml.ErrTypeMismatch))
		fmt.Printf("   %v\n", err)
	}
	fmt.Println()

	fmt.Println("4. Missing required key:")
	_, err = dml.Unmarshal[AppConfig]([]byte("int server.port = 1;\n"))
	fmt.Printf("   %v\n", err)
}
