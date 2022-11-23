package apirunner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-test/deep"
)

const (
	PassedString = "\033[1;32mPASSED (%dms)\033[0m"
	FailedString = "\033[1;31mFAILED (%dms)\033[0m"
	ErrorString  = "\033[1;31m%s\033[0m"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type TestRunner struct {
	suite      TestSuite
	config     Config
	suiteName  string
	httpClient HttpClient
}

// Create a 'TestRunner' for a given test file
func NewRunner(config Config, testFileName string) (TestRunner, error) {
	jsonFile, err := os.Open(testFileName)
	if err != nil {
		log.Printf("Error reading test file '%s': %v\n", testFileName, err)
		return TestRunner{}, err
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Printf("Error reading test file '%s': %v\n", testFileName, err)
		return TestRunner{}, err
	}

	var suite TestSuite
	err = json.Unmarshal(byteValue, &suite)
	if err != nil {
		log.Printf("Unable to parse test data in '%s': %v\n", testFileName, err)
		return TestRunner{}, err
	}
	return TestRunner{
		suite,
		config,
		testFileName,
		http.DefaultClient,
	}, nil
}

// Execute all tests and return results
func (runner TestRunner) Execute() TestSuiteResults {
	passed := make([]TestResult, 0)
	failed := make([]TestResult, 0)
	fmt.Printf("* '%s':\n", runner.suiteName)
	for _, test := range runner.suite.Tests {
		start := time.Now()
		result := runner.executeTest(test)
		duration := time.Since(start).Milliseconds()
		if result.Passed {
			passed = append(passed, result)
			fmt.Printf("\t%s %s\n", result.Name, fmt.Sprintf(PassedString, duration))
		} else {
			failed = append(failed, result)
			fmt.Printf("\t%s %s\n", result.Name, fmt.Sprintf(FailedString, duration))
			for _, err := range result.Errors {
				fmt.Printf("\t\t%s\n", fmt.Sprintf(ErrorString, err))
			}
		}
	}

	return TestSuiteResults{
		Passed: passed,
		Failed: failed,
	}
}

func (runner TestRunner) executeTest(test TestSpec) TestResult {
	testErrors := make([]string, 0)

	// Prep & make request
	var requestBody io.Reader
	if test.Request.Body == nil {
		requestBody = bytes.NewBuffer([]byte("{}"))
	} else {
		reqBodyBytes, err := json.Marshal(test.Request.Body)
		if err != nil {
			testErrors = append(testErrors, fmt.Sprintf("Invalid request body: %v", err))
			return Failed(test.Name, testErrors)
		}
		requestBody = bytes.NewBuffer(reqBodyBytes)
	}
	baseUrl := runner.config.BaseUrl
	if runner.suite.BaseUrl != "" {
		baseUrl = runner.suite.BaseUrl
	}
	if test.Request.BaseUrl != "" {
		baseUrl = test.Request.BaseUrl
	}
	req, err := http.NewRequest(test.Request.Method, baseUrl+test.Request.Url, requestBody)
	if err != nil {
		testErrors = append(testErrors, fmt.Sprintf("Unable to create request: %v", err))
		return Failed(test.Name, testErrors)
	}
	req.Header.Add("Authorization", runner.config.ApiKey)
	resp, err := runner.httpClient.Do(req)
	if err != nil {
		testErrors = append(testErrors, fmt.Sprintf("Error making request: %v", err))
		return Failed(test.Name, testErrors)
	}

	// Compare response statusCode
	statusCode := resp.StatusCode
	if statusCode != test.ExpectedResponse.StatusCode {
		testErrors = append(testErrors, fmt.Sprintf("Expected http %d but got http %d", test.ExpectedResponse.StatusCode, statusCode))
	}

	// Read response payload
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		testErrors = append(testErrors, fmt.Sprintf("Error reading response from server: %v", err))
		return Failed(test.Name, testErrors)
	}

	// Compare response payload
	expectedResponse := test.ExpectedResponse.Body
	// Confirm there is no response payload if that's what is expected
	if expectedResponse == nil {
		if len(body) != 0 {
			testErrors = append(testErrors, fmt.Sprintf("Expected response payload %s but got empty response", expectedResponse))
		}
		// No need to check anything else since no response was expected
		if len(testErrors) > 0 {
			return Failed(test.Name, testErrors)
		} else {
			return Passed(test.Name)
		}
	}

	// Otherwise, deep compare response payload to expected response payload
	var r interface{}
	err = json.Unmarshal(body, &r)
	if err != nil {
		testErrors = append(testErrors, fmt.Sprintf("Error parsing json response from server: %v", err))
		return Failed(test.Name, testErrors)
	}
	switch r.(type) {
	case map[string]interface{}:
		response := r.(map[string]interface{})
		runner.removeFieldsFromMap(response)
		expected := expectedResponse.(map[string]interface{})
		runner.removeFieldsFromMap(expected)
		differences := deep.Equal(response, expected)
		if len(differences) > 0 {
			testErrors = append(testErrors, differences...)
		}
	case []interface{}:
		response := r.([]interface{})
		expected := expectedResponse.([]interface{})
		if len(response) != len(expected) {
			testErrors = append(testErrors, "The number of array elements in response and expectedResponse don't match")
		} else {
			for i := range response {
				respObj := response[i].(map[string]interface{})
				runner.removeFieldsFromMap(respObj)
				expObj := expected[i].(map[string]interface{})
				runner.removeFieldsFromMap(expObj)
				differences := deep.Equal(respObj, expObj)
				if len(differences) > 0 {
					testErrors = append(testErrors, differences...)
				}
			}
		}
	default:
		differences := deep.Equal(r, expectedResponse)
		if len(differences) > 0 {
			testErrors = append(testErrors, differences...)
		}
	}
	if len(testErrors) > 0 {
		return Failed(test.Name, testErrors)
	}
	return Passed(test.Name)
}

func (runner TestRunner) removeFieldsFromMap(m map[string]interface{}) {
	for _, field := range runner.suite.IgnoredFields {
		delete(m, field)
	}
}
