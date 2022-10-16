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
	config := NewConfig("", "")
	testRunner, _ := NewRunner(config, "basicresponse.json")
	mockClient := MockHttpClient{}
	testRunner.httpClient = &mockClient
	mockClient.StatusCode = 200
	mockClient.Body = "{ \"id\": 1, \"name\": \"name\" }"

	passed, failed := testRunner.Execute()
	if len(passed) == 0 {
		t.Errorf("All tests should have passed.\n")
	}
	if len(failed) > 0 {
		for _, test := range failed {
			t.Errorf("Failed test result: [%s]\n", test.TabbedResult())
		}
	}
}

func TestListResponse(t *testing.T) {
	config := NewConfig("", "")
	testRunner, _ := NewRunner(config, "listresponse.json")
	mockClient := MockHttpClient{}
	testRunner.httpClient = &mockClient
	mockClient.StatusCode = 200
	mockClient.Body = "[{\"id\": 1, \"name\": \"name1\"},{\"id\": 2,\"name\": \"name2\"}]"

	passed, failed := testRunner.Execute()
	if len(passed) == 0 {
		t.Errorf("All tests should have passed.\n")
	}
	if len(failed) > 0 {
		for _, test := range failed {
			t.Errorf("Failed test result: [%s]\n", test.TabbedResult())
		}
	}
}
