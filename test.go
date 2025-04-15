package main

import (
    "fmt"
    "log"
    "github.com/tree-software-company/dml-go/dml"
)

func main() {
    data, err := dml.Parse("testdata/example.dml")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Greeting:", data["greeting"])
}
