// Copyright 2023 Forerunner Labs, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
