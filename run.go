// Copyright 2024 WorkOS, Inc.
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

package apirunner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type RunConfig struct {
	BaseUrl       string            `json:"baseUrl"`
	CustomHeaders map[string]string `json:"headers"`
	HttpClient    HttpClient
}

// Run executes all test files in 'testDir'. Returns true if all tests pass, false otherwise (including on err)
func Run(runConfigFilename string, testDir string, testFilenameMatchRegex *regexp.Regexp) (bool, error) {
	// Load and validate RunConfig
	configFile, err := os.Open(runConfigFilename)
	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("invalid config file: %s", runConfigFilename))
	}
	defer configFile.Close()
	configBytes, err := io.ReadAll(configFile)
	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("error reading %s", runConfigFilename))
	}
	var config RunConfig
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return false, errors.Wrap(err, "invalid run config")
	}
	// Use default httpclient to make requests
	config.HttpClient = http.DefaultClient

	// Find test files
	testFiles := make([]string, 0)
	err = filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") && testFilenameMatchRegex.MatchString(info.Name()) {
			fmt.Printf("Found '%s'\n", path)
			testFiles = append(testFiles, path)
		}

		return nil
	})

	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("Error reading dir: %s", testDir))
	}

	// Execute tests
	results := make([]TestSuiteResult, 0)
	start := time.Now()
	for _, testFile := range testFiles {
		suiteResult, err := ExecuteSuite(config, testFile, false)
		if err != nil {
			fmt.Printf("Error running tests for '%s': %v\n", testFile, err)
			continue
		}
		results = append(results, suiteResult)
	}
	execDuration := time.Since(start)

	total := 0
	numPassed := 0
	numFailed := 0
	numSkipped := 0
	for _, result := range results {
		total += result.TotalTests
		numFailed += len(result.Failed)
		numPassed += len(result.Passed)
		numSkipped += len(result.Skipped)
		if len(result.Failed) == 0 {
			continue
		}
		fmt.Printf("* Failures for '%s':\n", result.TestFilename)
		for _, failed := range result.Failed {
			fmt.Print(failed.Result())
		}
	}
	fmt.Printf("\nTotal: %d\nPassed: %d\nFailed: %d\nSkipped: %d\nDuration: %s\n", total, numPassed, numFailed, numSkipped, execDuration)
	if numFailed > 0 {
		return false, nil
	}
	return true, nil
}
