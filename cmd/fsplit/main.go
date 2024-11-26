package main

import (
	"log"
	"os"

	"github.com/nakario/fsplit"
)

func main() {
	// Check if the package path is provided
	if len(os.Args) < 2 {
		log.Fatalln("Usage: fsplit <package-path>")
	}

	packagePath := os.Args[1]
	if err := fsplit.RunFsplit(packagePath); err != nil {
		log.Fatalf("Error running fsplit: %v\n", err)
	}
}
