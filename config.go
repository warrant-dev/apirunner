package apirunner

type Config struct {
	BaseUrl string
	ApiKey  string
}

func NewConfig(baseUrl string, key string) Config {
	return Config{
		BaseUrl: baseUrl,
		ApiKey:  "ApiKey " + key,
	}
}
