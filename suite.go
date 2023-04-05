package apirunner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-test/deep"
	"github.com/pkg/errors"
)

const (
	PassedString = "\033[1;32mPASSED (%s)\033[0m"
	FailedString = "\033[1;31mFAILED (%s)\033[0m"
	ErrorString  = "\033[1;31m%s\033[0m"
)

// Mock-able HttpClient interface
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// A collection of tests defined within one test file
type TestSuite struct {
	spec     TestSuiteSpec
	config   RunConfig
	fileName string
}

// Spec defining the tests in a suite
type TestSuiteSpec struct {
	IgnoredFields []string   `json:"ignoredFields"`
	BaseUrl       string     `json:"baseUrl"`
	Tests         []TestSpec `json:"tests"`
}

// Spec defining a single test case
type TestSpec struct {
	Name             string           `json:"name"`
	Request          Request          `json:"request"`
	ExpectedResponse ExpectedResponse `json:"expectedResponse"`
}

// Request information for a single test case
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
type TestSuiteResult struct {
	Passed       []TestResult
	Failed       []TestResult
	TestFilename string
	TotalTests   int
}

// Result for an executed test case
type TestResult struct {
	Passed   bool
	Name     string
	Errors   []string
	Duration time.Duration
}

func Failed(name string, errors []string, duration time.Duration) TestResult {
	return TestResult{
		Passed:   false,
		Name:     name,
		Errors:   errors,
		Duration: duration,
	}
}

func Passed(name string, duration time.Duration) TestResult {
	return TestResult{
		Passed:   true,
		Name:     name,
		Errors:   nil,
		Duration: duration,
	}
}

func (result TestResult) ResultNoDetail() string {
	if result.Passed {
		return fmt.Sprintf("\t%s %s\n", result.Name, fmt.Sprintf(PassedString, result.Duration))
	} else {
		return fmt.Sprintf("\t%s %s\n", result.Name, fmt.Sprintf(FailedString, result.Duration))
	}
}

func (result TestResult) Result() string {
	resultString := result.ResultNoDetail()
	if !result.Passed {
		for _, err := range result.Errors {
			resultString = resultString + fmt.Sprintf("\t\t%s\n", fmt.Sprintf(ErrorString, err))
		}
	}
	return resultString
}

// Execute a test suite and print + return results
func ExecuteSuite(runConfig RunConfig, testFilename string, logFailureDetails bool) (TestSuiteResult, error) {
	// Read test suite spec
	jsonFile, err := os.Open(testFilename)
	if err != nil {
		return TestSuiteResult{}, errors.Wrap(err, fmt.Sprintf("error opening test file %s", testFilename))
	}
	defer jsonFile.Close()
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return TestSuiteResult{}, errors.Wrap(err, fmt.Sprintf("error reading test file %s", testFilename))
	}
	var suiteSpec TestSuiteSpec
	err = json.Unmarshal(byteValue, &suiteSpec)
	if err != nil {
		return TestSuiteResult{}, errors.Wrap(err, fmt.Sprintf("error parsing test data in %s", testFilename))
	}

	// Validate test suite spec (no duplicate tests, names must be alphanumeric without spaces)
	nameRegex := regexp.MustCompile(`^[a-zA-Z0-9]*$`)
	testNames := make(map[string]bool)
	for _, testSpec := range suiteSpec.Tests {
		if !nameRegex.MatchString(testSpec.Name) {
			return TestSuiteResult{}, fmt.Errorf("invalid test case name: '%s', must be alphanumeric without spaces", testSpec.Name)
		}
		if _, ok := testNames[testSpec.Name]; ok {
			return TestSuiteResult{}, fmt.Errorf("test case '%s' defined twice", testSpec.Name)
		}
		testNames[testSpec.Name] = true
	}

	// Execute test suite
	testSuite := TestSuite{
		suiteSpec,
		runConfig,
		testFilename,
	}
	passed := make([]TestResult, 0)
	failed := make([]TestResult, 0)
	totalTests := 0
	fmt.Printf("\n* '%s':\n", testSuite.fileName)
	// Memoized attrs map
	extractedFields := make(map[string]string)
	for _, test := range testSuite.spec.Tests {
		totalTests++
		result := testSuite.executeTest(test, extractedFields)
		if result.Passed {
			passed = append(passed, result)
		} else {
			failed = append(failed, result)
		}
		if logFailureDetails {
			fmt.Print(result.Result())
		} else {
			fmt.Print(result.ResultNoDetail())
		}
	}
	return TestSuiteResult{
		TotalTests:   totalTests,
		Passed:       passed,
		Failed:       failed,
		TestFilename: testSuite.fileName,
	}, nil
}

func (suite TestSuite) executeTest(test TestSpec, extractedFields map[string]string) TestResult {
	start := time.Now()
	testErrors := make([]string, 0)

	// Prep & make request
	var requestBody io.Reader
	if test.Request.Body == nil {
		requestBody = bytes.NewBuffer([]byte("{}"))
	} else {
		reqBodyBytes, err := json.Marshal(test.Request.Body)
		if err != nil {
			testErrors = append(testErrors, fmt.Sprintf("Invalid request body: %v", err))
			return Failed(test.Name, testErrors, time.Since(start))
		}
		requestBody = bytes.NewBuffer(reqBodyBytes)
	}
	baseUrl := suite.config.BaseUrl
	if suite.spec.BaseUrl != "" {
		baseUrl = suite.spec.BaseUrl
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
		return Failed(test.Name, testErrors, time.Since(start))
	}
	for k, v := range suite.config.CustomHeaders {
		req.Header.Add(k, v)
	}
	resp, err := suite.config.HttpClient.Do(req)
	if err != nil {
		testErrors = append(testErrors, fmt.Sprintf("Error making request: %v", err))
		return Failed(test.Name, testErrors, time.Since(start))
	}

	// Compare response statusCode
	statusCode := resp.StatusCode
	if statusCode != test.ExpectedResponse.StatusCode {
		testErrors = append(testErrors, fmt.Sprintf("Expected http %d but got http %d", test.ExpectedResponse.StatusCode, statusCode))
	}

	// Read response payload
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		testErrors = append(testErrors, fmt.Sprintf("Error reading response from server: %v", err))
		return Failed(test.Name, testErrors, time.Since(start))
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
			return Failed(test.Name, testErrors, time.Since(start))
		} else {
			return Passed(test.Name, time.Since(start))
		}
	}

	// Otherwise, deep compare response payload to expected response payload
	var r interface{}
	err = json.Unmarshal(body, &r)
	if err != nil {
		testErrors = append(testErrors, fmt.Sprintf("Error parsing json response from server: %v", err))
		return Failed(test.Name, testErrors, time.Since(start))
	}
	switch r.(type) {
	case map[string]interface{}:
		differences, err := suite.compareObjects(r.(map[string]interface{}), expectedResponse.(map[string]interface{}), extractedFields, test.Name)
		if err != nil {
			testErrors = append(testErrors, fmt.Sprintf("Error comparing actual and expected responses: %v", err))
		}

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
				differences, err := suite.compareObjects(response[i].(map[string]interface{}), expected[i].(map[string]interface{}), extractedFields, fmt.Sprintf("%s[%d]", test.Name, i))
				if err != nil {
					testErrors = append(testErrors, fmt.Sprintf("Error comparing actual and expected responses: %v", err))
				}

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
		return Failed(test.Name, testErrors, time.Since(start))
	}
	return Passed(test.Name, time.Since(start))
}

func (suite TestSuite) compareObjects(obj map[string]interface{}, expectedObj map[string]interface{}, extractedFields map[string]string, objPrefix string) ([]string, error) {
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

	// Deep compare the objects and return any errors,
	// ignoring any errors that match an ignored field.
	//
	// NOTE: This approach is brittle as it assumes the
	// github.com/go-test/deep package's Equal method
	// continues to return errors in the expected format.
	diffs := make([]string, 0)
	allDiffs := deep.Equal(obj, expectedObj)
	ignoredFieldsMatchExpr := fmt.Sprintf(`\[%s\]$`, strings.Join(suite.spec.IgnoredFields, `\]|\[`))
	for _, diff := range allDiffs {
		field, _, found := strings.Cut(diff, ":")
		if !found {
			return nil, fmt.Errorf("invalid diff %s returned by deep.Equal", diff)
		}

		// ignore errors that match an ignored field
		matched, err := regexp.MatchString(ignoredFieldsMatchExpr, field)
		if err != nil {
			return nil, err
		}

		if !matched {
			diffs = append(diffs, diff)
		}
	}

	return diffs, nil
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
