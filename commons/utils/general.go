package utils

import (
	"log"
	"os"
)

func GetEnv(key string) string {
	value, exists := os.LookupEnv(key); 

	if(!exists || value == "") {
		log.Fatalf("Environment variable %v is required", key)
	}

	return value
}