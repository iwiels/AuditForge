package main

import (
	"fmt"
	"os"

	"orquestador-auditor/internal/app"
)

var version = "0.1.0-dev"

func main() {
	if err := app.Run(version); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
