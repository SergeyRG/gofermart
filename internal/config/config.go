package config

type Config struct {
	ServerAddress string
}

func NewConfig() (Config, error) {
	return Config{ServerAddress: ":8080"}, nil
}
