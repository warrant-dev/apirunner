package apirunner

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

type MockHttpClient struct {
	StatusCode int
	Body       string
}

func (c *MockHttpClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: c.StatusCode,
		Body:       ioutil.NopCloser(strings.NewReader(c.Body)),
	}, nil
}

func TestBasicResponse(t *testing.T) {
	testRunner, _ := NewRunner(Config{
		TestFileName:  "basicresponse.json",
		BaseUrl:       "",
		CustomHeaders: nil,
	})
	mockClient := MockHttpClient{}
	testRunner.httpClient = &mockClient
	mockClient.StatusCode = 200
	mockClient.Body = "{ \"id\": 1, \"name\": \"name\" }"

	results := testRunner.Execute()

	if len(results.Passed) == 0 {
		t.Errorf("All tests should have passed.\n")
	}
	if len(results.Failed) > 0 {
		for _, test := range results.Failed {
			t.Errorf("Failed test result: [%s]\n", test.Serialize())
		}
	}
}

func TestListResponse(t *testing.T) {
	testRunner, _ := NewRunner(Config{
		TestFileName:  "listresponse.json",
		BaseUrl:       "",
		CustomHeaders: nil,
	})
	mockClient := MockHttpClient{}
	testRunner.httpClient = &mockClient
	mockClient.StatusCode = 200
	mockClient.Body = "[{\"id\": 1, \"name\": \"name1\"},{\"id\": 2,\"name\": \"name2\"}]"

	results := testRunner.Execute()
	if len(results.Passed) == 0 {
		t.Errorf("All tests should have passed.\n")
	}
	if len(results.Failed) > 0 {
		for _, test := range results.Failed {
			t.Errorf("Failed test result: [%s]\n", test.Serialize())
		}
	}
}
