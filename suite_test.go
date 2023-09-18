// Copyright 2023 Forerunner Labs, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	Header     map[string][]string
}

func (c *MockHttpClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: c.StatusCode,
		Body:       io.NopCloser(strings.NewReader(c.Body)),
		Header:     c.Header,
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
	respHeaders := make(map[string][]string)
	respHeaders["Warrant-Token"] = []string{"asdf"}
	mockClient.Header = respHeaders
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

	if len(results.Passed) != 1 && len(results.Failed) != 1 && len(results.Skipped) != 0 {
		t.Errorf("Expected 1 Passed, 1 Failed, 0 Skipped.")
	}

	if !strings.Contains(results.Failed[0].Result(), "missing template value for var: 'test1.userIdWrongVar'") {
		t.Errorf("Expected failure result to contain string: 'missing template value for var: 'test1.userIdWrongVar''")
	}
}
