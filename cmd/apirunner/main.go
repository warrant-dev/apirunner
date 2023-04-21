package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/warrant-dev/apirunner"
)

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Printf("Invalid args")
		os.Exit(1)
	}
	testDir := os.Args[1]
	configFile := filepath.Join(testDir, "apirunner.conf")
	if len(os.Args) == 3 {
		// If configFile passed in, use that
		configFile = os.Args[2]
	}
	passed, err := apirunner.Run(configFile, testDir)
	if err != nil {
		fmt.Printf("Error executing tests: %v\n", err)
		os.Exit(1)
	}
	if !passed {
		os.Exit(1)
	}
	os.Exit(0)
}
