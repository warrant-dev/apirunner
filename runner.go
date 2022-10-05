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

type TestRunner struct {
	tests     Tests
	config    Config
	suiteName string
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
	}, nil
}

func (runner TestRunner) Execute() {
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
	req, err := http.NewRequest(test.Request.Method, runner.config.ServerEndpoint+test.Request.Url, requestBody)
	if err != nil {
		return Failed(test.Name, nil), err
	}
	req.Header.Add("Authorization", runner.config.ApiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Failed(test.Name, nil), err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Failed(test.Name, nil), err
	}
	var response map[string]interface{}
	json.Unmarshal(body, &response)
	runner.removeFieldsFromMap(response)

	// Compare statusCode and deep compare response payload
	failures := make([]string, 0)
	statusCode := resp.StatusCode
	if statusCode != test.ExpectedResponse.StatusCode {
		failures = append(failures, fmt.Sprintf("Expected http %d but got http %d", test.ExpectedResponse.StatusCode, statusCode))
	}
	expectedResponse := test.ExpectedResponse.Body
	runner.removeFieldsFromMap(expectedResponse)
	differences := deep.Equal(response, expectedResponse)
	if len(differences) > 0 {
		failures = append(failures, differences...)
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
