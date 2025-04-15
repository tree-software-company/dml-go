package main

import (
    "fmt"
    "log"

    "github.com/tree-software-company/dml-go/dml"
)

func main() {
    result, err := dml.Parse("testdata/example.dml")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Greeting:", result["greeting"])
}
