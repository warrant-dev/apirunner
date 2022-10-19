package apirunner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/go-test/deep"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type TestRunner struct {
	tests      Tests
	config     Config
	suiteName  string
	httpClient HttpClient
}

func NewRunner(config Config, testFileName string) (TestRunner, error) {
	jsonFile, err := os.Open(testFileName)
	if err != nil {
		fmt.Println(err)
		return TestRunner{}, err
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var tests Tests
	json.Unmarshal(byteValue, &tests)
	return TestRunner{
		tests,
		config,
		testFileName,
		http.DefaultClient,
	}, nil
}

func (runner TestRunner) Execute() ([]TestResult, []TestResult) {
	passed := make([]TestResult, 0)
	failed := make([]TestResult, 0)
	for _, test := range runner.tests.Tests {
		result, _ := runner.executeTest(test)
		if result.Passed {
			passed = append(passed, result)
		} else {
			failed = append(failed, result)
		}
	}

	fmt.Printf("Results from %s:\n", runner.suiteName)
	for _, result := range passed {
		result.PrintResult()
	}
	for _, result := range failed {
		result.PrintResult()
	}
	return passed, failed
}

func (runner TestRunner) executeTest(test Test) (TestResult, error) {
	// Prep & make request
	var requestBody io.Reader
	if test.Request.Body == nil {
		requestBody = bytes.NewBuffer([]byte("{}"))
	} else {
		reqBodyBytes, err := json.Marshal(test.Request.Body)
		if err != nil {
			return Failed(test.Name, nil), err
		}
		requestBody = bytes.NewBuffer(reqBodyBytes)
	}
	baseUrl := runner.config.BaseUrl
	if runner.tests.BaseUrl != "" {
		baseUrl = runner.tests.BaseUrl
	}
	if test.Request.BaseUrl != "" {
		baseUrl = test.Request.BaseUrl
	}
	req, err := http.NewRequest(test.Request.Method, baseUrl+test.Request.Url, requestBody)
	if err != nil {
		return Failed(test.Name, nil), err
	}
	req.Header.Add("Authorization", runner.config.ApiKey)
	resp, err := runner.httpClient.Do(req)
	if err != nil {
		return Failed(test.Name, nil), err
	}

	// Compare statusCode
	failures := make([]string, 0)
	statusCode := resp.StatusCode
	if statusCode != test.ExpectedResponse.StatusCode {
		failures = append(failures, fmt.Sprintf("Expected http %d but got http %d", test.ExpectedResponse.StatusCode, statusCode))
	}

	// Deep compare response payload
	expectedResponse := test.ExpectedResponse.Body

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Failed(test.Name, nil), err
	}
	var r interface{}
	json.Unmarshal(body, &r)
	switch r.(type) {
	case map[string]interface{}:
		response := r.(map[string]interface{})
		runner.removeFieldsFromMap(response)
		expected := expectedResponse.(map[string]interface{})
		runner.removeFieldsFromMap(expected)
		differences := deep.Equal(response, expected)
		if len(differences) > 0 {
			failures = append(failures, differences...)
		}
	case []interface{}:
		response := r.([]interface{})
		expected := expectedResponse.([]interface{})
		if len(response) != len(expected) {
			failures = append(failures, "The number of array elements in response and expectedResponse don't match")
		} else {
			for i := range response {
				respObj := response[i].(map[string]interface{})
				runner.removeFieldsFromMap(respObj)
				expObj := expected[i].(map[string]interface{})
				runner.removeFieldsFromMap(expObj)
				differences := deep.Equal(respObj, expObj)
				if len(differences) > 0 {
					failures = append(failures, differences...)
				}
			}
		}
	default:
		differences := deep.Equal(r, expectedResponse)
		if len(differences) > 0 {
			failures = append(failures, differences...)
		}
	}

	if len(failures) > 0 {
		return Failed(test.Name, failures), nil
	}
	return Passed(test.Name), nil
}

func (runner TestRunner) removeFieldsFromMap(m map[string]interface{}) {
	for _, field := range runner.tests.IgnoredFields {
		delete(m, field)
	}
}
