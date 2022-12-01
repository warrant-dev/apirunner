package apirunner

import (
	"fmt"
)

// A collection of tests to execute that share config
type TestSuite struct {
	IgnoredFields []string   `json:"ignoredFields"`
	BaseUrl       string     `json:"baseUrl"`
	Tests         []TestSpec `json:"tests"`
}

// A spec defining a single test case
type TestSpec struct {
	Name             string           `json:"name"`
	Request          Request          `json:"request"`
	ExpectedResponse ExpectedResponse `json:"expectedResponse"`
}

// Request information for a test case
type Request struct {
	Method  string      `json:"method"`
	BaseUrl string      `json:"baseUrl"`
	Url     string      `json:"url"`
	Body    interface{} `json:"body"`
}

// Expected test case response
type ExpectedResponse struct {
	StatusCode int         `json:"statusCode"`
	Body       interface{} `json:"body"`
}

// Results for an executed TestSuite
type TestSuiteResults struct {
	Passed []TestResult
	Failed []TestResult
}

// A result for an executed test case
type TestResult struct {
	Passed bool
	Name   string
	Errors []string
}

func Failed(name string, errors []string) TestResult {
	return TestResult{
		Passed: false,
		Name:   name,
		Errors: errors,
	}
}

func Passed(name string) TestResult {
	return TestResult{
		Passed: true,
		Name:   name,
		Errors: nil,
	}
}

// Serialize test result (pass or fail) as string (including errors)
func (result TestResult) Serialize() []string {
	resultString := make([]string, 0)
	if result.Passed {
		resultString = append(resultString, fmt.Sprintf("%s:\tPASSED\n", result.Name))
	} else {
		resultString = append(resultString, fmt.Sprintf("%s:\tFAILED\n", result.Name))
		for _, err := range result.Errors {
			resultString = append(resultString, fmt.Sprintf("\t%s\n", err))
		}
	}
	return resultString
}
