package apirunner

import (
	"io"
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
		Body:       io.NopCloser(strings.NewReader(c.Body)),
	}, nil
}

func TestBasicResponse(t *testing.T) {
	mockClient := MockHttpClient{}
	mockClient.StatusCode = 200
	mockClient.Body = "{ \"id\": 1, \"name\": \"name\" }"

	results, _ := ExecuteSuite(RunConfig{
		BaseUrl:       "",
		CustomHeaders: nil,
		HttpClient:    &mockClient,
	}, "basicresponse.json", true)

	if len(results.Passed) == 0 {
		t.Errorf("All tests should have passed.\n")
	}
	if len(results.Failed) > 0 {
		for _, test := range results.Failed {
			t.Errorf("Failed test result: [%s]\n", test.Result())
		}
	}
}

func TestListResponse(t *testing.T) {
	mockClient := MockHttpClient{}
	mockClient.StatusCode = 200
	mockClient.Body = "[{\"id\": 1, \"name\": \"name1\"},{\"id\": 2,\"name\": \"name2\"}]"
	results, _ := ExecuteSuite(RunConfig{
		BaseUrl:       "",
		CustomHeaders: nil,
		HttpClient:    &mockClient,
	}, "listresponse.json", true)

	if len(results.Passed) == 0 {
		t.Errorf("All tests should have passed.\n")
	}
	if len(results.Failed) > 0 {
		for _, test := range results.Failed {
			t.Errorf("Failed test result: [%s]\n", test.Result())
		}
	}
}
