package config

type Config struct {
	ServerAddress        string
	DBDSN                string
	AccrualSystemAddress string
}

func NewConfig() (Config, error) {
	return Config{
		ServerAddress:        ":8080",
		DBDSN:                "postgres://test:test@localhost:5432/test?sslmode=disable",
		AccrualSystemAddress: "http://127.0.0.1:8081",
	}, nil
}
