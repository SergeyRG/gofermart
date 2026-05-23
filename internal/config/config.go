package config

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	ServerAddress        string
	DBDSN                string
	AccrualSystemAddress string
	SecretKey            string
}

func validateURL(val string) error {
	uParsed, err := url.ParseRequestURI(val)
	if err != nil {
		return fmt.Errorf("flag -b value must be in form http[s]://host:port, "+
			"received: %s", val)
	}

	if uParsed.Scheme != "http" && uParsed.Scheme != "https" {
		return fmt.Errorf("flag -b value must be in form http[s]://host:port, "+
			"received: %s", val)
	}

	if uParsed.User != nil {
		return fmt.Errorf("flag -b value must be in form http[s]://host:port, "+
			"received: %s", val)
	}

	if uParsed.Path != "" {
		return fmt.Errorf("flag -b value must be in form http[s]://host:port, "+
			"received: %s", val)
	}

	if err := validatePortString(uParsed.Port()); err != nil {
		return err
	}

	return nil
}

func validatePortString(port string) error {

	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("incorrect port")
	}
	return nil

}

func validateServerAddress(val string) error {
	_, port, err := net.SplitHostPort(val)
	if err != nil {
		return fmt.Errorf("flag -a value must be in form host:port, "+
			"received: %s", val)
	}

	if err := validatePortString(port); err != nil {
		return errors.New("port value is specified outside the acceptable range")
	}
	return nil
}

func NewConfig() (Config, error) {
	defKey := string("DEFAULT_SECRET_KEY")
	SecretKey := &defKey

	ServerAddress := flag.String(
		"a", ":8080", "address and port to run server")
	AccrualSystemAddress := flag.String(
		"r", "http://127.0.0.1:8081", "адрес системы расчёта начислений")
	DBDSN := flag.String("d", "postgres://test:test@localhost:5432/test?sslmode=disable", "DSN to connect to the database.")

	flag.Parse()

	if val, exist := os.LookupEnv("RUN_ADDRESS"); exist {
		*ServerAddress = val
	}
	if val, exist := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); exist {
		*AccrualSystemAddress = val
	}
	if val, exist := os.LookupEnv("DATABASE_URI"); exist {
		*DBDSN = val
	}
	if val, exist := os.LookupEnv("SECRET_KEY"); exist {
		*SecretKey = val
	} else {
		log.Printf("Используется ключ по умолчанию")
	}

	if err := validateServerAddress(*ServerAddress); err != nil {
		return Config{}, err
	}

	if err := validateURL(*AccrualSystemAddress); err != nil {
		return Config{}, err
	}

	return Config{
			ServerAddress:        *ServerAddress,
			AccrualSystemAddress: *AccrualSystemAddress,
			DBDSN:                *DBDSN,
			SecretKey:            *SecretKey,
		},
		nil
}
