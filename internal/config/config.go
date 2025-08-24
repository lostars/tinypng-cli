package config

import "os"

var APIKey string

type Config struct {
}

func GetAPIKey() string {
	// read from command flag first
	if APIKey != "" {
		return APIKey
	}
	// read from env
	key := os.Getenv("TINYPNG_API_KEY")
	if key != "" {
		return key
	}
	panic("tinypng api key not set")
}
