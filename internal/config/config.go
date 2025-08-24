package config

import (
	"os"
)

var C Config

type Config struct {
	APIKey  string
	Timeout int64
}

func GetAPIKey() string {
	// read from command flag first
	if C.APIKey != "" {
		return C.APIKey
	}
	// read from env
	key := os.Getenv("TINYPNG_API_KEY")
	if key != "" {
		return key
	}
	panic("tinypng api key not set")
}
