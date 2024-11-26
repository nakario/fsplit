package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nakario/fsplit"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s <package-path>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	// Check if the package path is provided as a positional argument
	if flag.NArg() < 1 {
		flag.Usage()
		log.Fatalln("Error: package path is required")
	}

	packagePath := flag.Arg(0)
	if err := fsplit.RunFsplit(packagePath); err != nil {
		log.Fatalf("Error running fsplit: %v\n", err)
	}
}
