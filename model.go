package apirunner

type Tests struct {
	IgnoredFields []string `json:"ignoredFields"`
	Tests         []Test   `json:"tests"`
}

type Test struct {
	Name             string           `json:"name"`
	Request          Request          `json:"request"`
	ExpectedResponse ExpectedResponse `json:"expectedResponse"`
}

type Request struct {
	Method string                 `json:"method"`
	Url    string                 `json:"url"`
	Body   map[string]interface{} `json:"body"`
}

type ExpectedResponse struct {
	StatusCode int         `json:"statusCode"`
	Body       interface{} `json:"body"`
}
