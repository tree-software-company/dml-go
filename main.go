package main

import (
	"fmt"
	"log"

	"github.com/tree-software-company/dml-go"
)

func main() {
	data, err := dml.Parse("test.dml")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Config loaded:", data)
}
