package apirunner

type Config struct {
	ServerEndpoint string
	ApiKey         string
}

func NewConfig(serverEndpoint string, key string) Config {
	return Config{
		ServerEndpoint: serverEndpoint,
		ApiKey:         "ApiKey " + key,
	}
}
