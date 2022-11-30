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
	"regexp"
	"strings"
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

type Config struct {
	BaseUrl       string
	CustomHeaders map[string]string
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

	// Validate test names (no duplicates, must be alphanumeric without spaces)
	nameRegex := regexp.MustCompile(`^[a-zA-Z0-9]*$`)
	testNames := make(map[string]bool)
	for _, testSpec := range suite.Tests {
		if !nameRegex.MatchString(testSpec.Name) {
			return TestRunner{}, fmt.Errorf("Invalid test case name: '%s', must be alphanumeric without spaces", testSpec.Name)
		}
		if _, ok := testNames[testSpec.Name]; ok {
			return TestRunner{}, fmt.Errorf("Test case '%s' defined twice", testSpec.Name)
		}
		testNames[testSpec.Name] = true
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
	extractedFields := make(map[string]string)
	for _, test := range runner.suite.Tests {
		start := time.Now()
		result := runner.executeTest(test, extractedFields)
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

func (runner TestRunner) executeTest(test TestSpec, extractedFields map[string]string) TestResult {
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
	requestUrlParts := strings.Split(test.Request.Url, "/")
	var requestUrl string
	for _, part := range requestUrlParts {
		if part != "" {
			requestUrl += "/" + getTemplateValIfPresent(part, extractedFields)
		}
	}
	req, err := http.NewRequest(test.Request.Method, baseUrl+requestUrl, requestBody)
	if err != nil {
		testErrors = append(testErrors, fmt.Sprintf("Unable to create request: %v", err))
		return Failed(test.Name, testErrors)
	}
	for k, v := range runner.config.CustomHeaders {
		req.Header.Add(k, v)
	}
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
		differences := runner.compareObjects(r.(map[string]interface{}), expectedResponse.(map[string]interface{}), extractedFields, test.Name)
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
				differences := runner.compareObjects(response[i].(map[string]interface{}), expected[i].(map[string]interface{}), extractedFields, fmt.Sprintf("%s[%d]", test.Name, i))
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
		// Append raw server response payload to errors for easier debugging
		testErrors = append(testErrors, fmt.Sprintf("Full response payload from server: %s", string(body)))
		return Failed(test.Name, testErrors)
	}
	return Passed(test.Name)
}

func (runner TestRunner) compareObjects(obj map[string]interface{}, expectedObj map[string]interface{}, extractedFields map[string]string, objPrefix string) []string {
	// Remove all ignored fields from obj and expectedObj so they aren't compared
	for _, field := range runner.suite.IgnoredFields {
		delete(obj, field)
		delete(expectedObj, field)
	}
	// Track all new field values from response obj
	for k, v := range obj {
		switch str := v.(type) {
		case string:
			extractedFields[objPrefix+"."+k] = str
		}
	}
	// Replace any template strings in expectedObj with values from extracted fields
	for k, v := range expectedObj {
		switch str := v.(type) {
		case string:
			if isTemplateString(str) {
				expectedObj[k] = getTemplateValIfPresent(str, extractedFields)
			}
		}
	}
	// Deep compare the objects and return all errors
	return deep.Equal(obj, expectedObj)
}

// Returns true if 's' is a template string of the format '{{ value }}'
func isTemplateString(s string) bool {
	return strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}")
}

// If 's' is a template string of the format "{{ value }}", resolve its value and return it. Else return original string.
func getTemplateValIfPresent(s string, extractedFields map[string]string) string {
	if isTemplateString(s) {
		key := strings.TrimSpace(s[2 : len(s)-2])
		if replacementValue, ok := extractedFields[key]; ok {
			return replacementValue
		}
	}
	return s
}
