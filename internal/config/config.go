package config

type Config struct {
	ServerAddress string
	DBDSN         string
}

func NewConfig() (Config, error) {
	return Config{
		ServerAddress: ":8080",
		DBDSN:         "postgres://test:test@localhost:5432/test?sslmode=disable",
	}, nil
}
