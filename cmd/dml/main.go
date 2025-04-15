package main

import (
    "fmt"
    "log"
    "os"

    "github.com/tree-software-company/dml-go/dml"
)

func main() {
    if len(os.Args) < 3 || os.Args[1] != "read" {
        fmt.Println("Usage: dml read <file.dml>")
        os.Exit(1)
    }

    result, err := dml.Parse(os.Args[2])
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("📄 Output:\n%v\n", result)
}
