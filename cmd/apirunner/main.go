package main

import (
	"fmt"
	"os"

	"github.com/warrant-dev/apirunner"
)

const RunConfigFilename = "apirunner.conf"

func main() {
	dirArg := os.Args[1]
	passed, err := apirunner.Run(RunConfigFilename, dirArg)
	if err != nil {
		fmt.Printf("Error executing tests: %v\n", err)
		os.Exit(1)
	}
	if !passed {
		os.Exit(1)
	}
	os.Exit(0)
}
