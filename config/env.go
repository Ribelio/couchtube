package config

import (
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

var (
	port         string
	dbFilePath   string
	jsonFilePath string
	readonly     bool
	once         sync.Once
)

func init() {
	once.Do(func() {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found, relying on system environment variables")
		}

		port = getEnv("PORT", "8363")
		dbFilePath = getEnvAsPath("DATABASE_FILE_PATH", "couchtube.db")
		jsonFilePath = getEnvAsPath("JSON_FILE_PATH", "videos.json")
		readonly = getEnvAsBool("READONLY_MODE", false)
	})
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("Warning: unable to parse boolean from %s; using default: %v", key, fallback)
			return fallback
		}
		return boolValue
	}
	return fallback
}

func getEnvAsPath(key, fallback string) string {
	value := getEnv(key, fallback)
	if value[0] != '/' {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		value = wd + "/" + value
	}
	return value
}

func GetPort() string {
	return port
}

func GetDBFilePath() string {
	return dbFilePath
}

func GetJSONFilePath() string {
	return jsonFilePath
}

func GetReadonlyMode() bool {
	return readonly
}
