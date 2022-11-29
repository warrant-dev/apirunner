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
			newPart := part
			if strings.HasPrefix(part, "{{") && strings.HasSuffix(part, "}}") {
				key := strings.TrimSpace(part[2 : len(part)-2])
				if replacementValue, ok := extractedFields[key]; ok {
					newPart = replacementValue
				}
			}
			requestUrl += "/" + newPart
		}
	}
	fmt.Printf("Request url: '%s'\n", requestUrl)
	req, err := http.NewRequest(test.Request.Method, baseUrl+requestUrl, requestBody)
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
		// response := r.(map[string]interface{})
		// runner.removeIgnoredFields(response)
		// expected := expectedResponse.(map[string]interface{})
		// runner.removeIgnoredFields(expected)
		// runner.extractFieldsFromResponse(response, extractedFields, test.Name)
		// runner.fillTemplateValues(expected, extractedFields, test.Name)
		// differences := deep.Equal(response, expected)
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
				// respObj := response[i].(map[string]interface{})
				// runner.removeIgnoredFields(respObj)
				// expObj := expected[i].(map[string]interface{})
				// runner.removeIgnoredFields(expObj)
				// runner.extractFieldsFromResponse(respObj, extractedFields, fmt.Sprintf("%s[%d]", test.Name, i))
				// runner.fillTemplateValues(expObj, extractedFields, fmt.Sprintf("%s[%d]", test.Name, i))
				// differences := deep.Equal(respObj, expObj)
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
		return Failed(test.Name, testErrors)
	}
	return Passed(test.Name)
}

func (runner TestRunner) compareObjects(obj map[string]interface{}, expected map[string]interface{}, extractedFields map[string]string, testName string) []string {
	//response := r.(map[string]interface{})
	//runner.removeIgnoredFields(obj)
	//expected := expectedResponse.(map[string]interface{})
	//runner.removeIgnoredFields(expected)
	for _, field := range runner.suite.IgnoredFields {
		delete(obj, field)
		delete(expected, field)
	}
	//runner.extractFieldsFromResponse(obj, extractedFields, testName)
	for k, v := range obj {
		extractedFields[testName+"."+k] = v.(string)
	}
	//runner.fillTemplateValues(expected, extractedFields, testName)
	for k, v := range expected {
		val := v.(string)
		if strings.HasPrefix(val, "{{") && strings.HasSuffix(val, "}}") {
			key := strings.TrimSpace(val[2 : len(val)-2])
			if replacementValue, ok := extractedFields[key]; ok {
				expected[k] = replacementValue
			}
			fmt.Printf("regex match: '%s'\n", key)
		}
	}
	return deep.Equal(obj, expected)
	// if len(differences) > 0 {
	// 	testErrors = append(testErrors, differences...)
	// }
}

// func (runner TestRunner) removeIgnoredFields(m map[string]interface{}) {
// 	for _, field := range runner.suite.IgnoredFields {
// 		delete(m, field)
// 	}
// }

// func (runner TestRunner) extractFieldsFromResponse(m map[string]interface{}, extractedFieldsMap map[string]string, fieldPrefix string) {
// 	for k, v := range m {
// 		extractedFieldsMap[fieldPrefix+"."+k] = v.(string)
// 	}
// }

// func (runner TestRunner) fillTemplateValues(m map[string]interface{}, extractedFieldsMap map[string]string, fieldPrefix string) {
// 	for k, v := range m {
// 		val := v.(string)
// 		if strings.HasPrefix(val, "{{") && strings.HasSuffix(val, "}}") {
// 			key := strings.TrimSpace(val[2 : len(val)-2])
// 			if replacementValue, ok := extractedFieldsMap[key]; ok {
// 				m[k] = replacementValue
// 			}
// 			fmt.Printf("regex match: '%s'\n", key)
// 		}
// 	}
// }
