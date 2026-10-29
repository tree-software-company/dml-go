package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/tree-software-company/dml-go/dml"
)

type Database struct {
	Host string `dml:"host" validate:"required"`
	Port int    `dml:"port" validate:"min=1,max=65535"`
}

type Server struct {
	Name  string   `dml:"name" validate:"required,min=3"`
	Mode  string   `dml:"mode" validate:"oneof=dev staging prod"`
	Ports []int    `dml:"ports" validate:"min=1,dive,min=1,max=65535"`
	Tags  []string `dml:"tags" validate:"max=3,dive,required"`
}

type AppConfig struct {
	Name     string    `dml:"name" validate:"required,min=3"`
	Server   Server    `dml:"server"`
	Database Database  `dml:"database"`
	Replica  *Database `dml:"replica"`
}

const validSource = `
string name = "BeautyApp";
string server.name = "api";
string server.mode = "prod";
list server.ports = [80, 443];
list server.tags = ["public", "eu"];
string database.host = "localhost";
int database.port = 5432;
`

const invalidSource = `
string name = "ab";
string server.name = "api";
string server.mode = "testing";
list server.ports = [80, 70000, 0];
list server.tags = ["public", "", "eu", "extra"];
int database.port = 0;
string replica.host = "replica.local";
int replica.port = 99999;
`

func main() {
	fmt.Println("=== DML Struct Validation ===")
	fmt.Println()

	cfg, err := dml.Unmarshal[AppConfig]([]byte(validSource))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("1. A valid config passes:")
	fmt.Printf("   name=%s mode=%s ports=%v db=%s:%d\n", cfg.Name, cfg.Server.Mode, cfg.Server.Ports, cfg.Database.Host, cfg.Database.Port)
	fmt.Println()

	fmt.Println("2. Every violation is reported with its path:")
	_, err = dml.Unmarshal[AppConfig]([]byte(invalidSource))
	var validationErr *dml.ValidationError
	if errors.As(err, &validationErr) {
		for _, field := range validationErr.Fields {
			fmt.Printf("   %-18s rule=%s param=%q\n", field.Path, field.Rule, field.Param)
		}
	}
	fmt.Println()

	fmt.Println("3. The error matches the sentinel and never contains values:")
	fmt.Printf("   validation=%t\n", errors.Is(err, dml.ErrValidation))
	fmt.Printf("   %v\n", err)
	fmt.Println()

	fmt.Println("4. A missing section still has its required keys checked:")
	_, err = dml.Unmarshal[AppConfig]([]byte("string name = \"BeautyApp\";\n"))
	fmt.Printf("   %v\n", err)
	fmt.Println()

	fmt.Println("5. Validate every live reload:")
	fmt.Println("   dml.Watch(path, dml.WatchOptions{Validate: func(c *dml.Config) error {")
	fmt.Println("       _, err := dml.Decode[AppConfig](c)")
	fmt.Println("       return err")
	fmt.Println("   }})")
}
