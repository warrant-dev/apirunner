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

func TestIgnoredFields(t *testing.T) {
	mockClient := MockHttpClient{}
	mockClient.StatusCode = 200
	mockClient.Body = "[{\"id\": 1, \"name\": \"name1\", \"nested\": {\"hello\": 1, \"createdAt\": \"2023-04-05T12:38:54.038Z\"}, \"createdAt\": \"2023-04-05T12:38:54.038Z\"},{\"id\": 2,\"name\": \"name2\", \"nested\": {\"hello\": 0, \"createdAt\": \"2023-04-05T12:38:54.036226Z\"}, \"createdAt\": \"2023-04-05T12:38:54.036226Z\"}]"
	results, _ := ExecuteSuite(RunConfig{
		BaseUrl:       "",
		CustomHeaders: nil,
		HttpClient:    &mockClient,
	}, "ignoredfieldsresponse.json", true)

	if len(results.Passed) == 0 {
		t.Errorf("All tests should have passed.\n")
	}
	if len(results.Failed) > 0 {
		for _, test := range results.Failed {
			t.Errorf("Failed test result: [%s]\n", test.Result())
		}
	}
}

func TestTemplateVars(t *testing.T) {
	mockClient := MockHttpClient{}
	mockClient.StatusCode = 200
	mockClient.Body = "{\"userId\": \"user_1\"}"
	results, _ := ExecuteSuite(RunConfig{
		BaseUrl:       "",
		CustomHeaders: nil,
		HttpClient:    &mockClient,
	}, "templatevars.json", true)

	if len(results.Passed) == 0 {
		t.Errorf("All tests should have passed.\n")
	}
	if len(results.Failed) > 0 {
		for _, test := range results.Failed {
			t.Errorf("Failed test result: [%s]\n", test.Result())
		}
	}
}
