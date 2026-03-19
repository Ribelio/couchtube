package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

var (
	port                string
	dbFilePath          string
	defaultChannelsPath string
	editorMode          string
	once                sync.Once
)

func init() {
	once.Do(func() {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found, relying on system environment variables")
		}

		port = getEnv("PORT", "8363")
		dbFilePath = getEnv("DATABASE_FILE_PATH", "couchtube.db")
		defaultChannelsPath = getEnv("DEFAULT_CHANNELS_PATH", "/videos.json")
		editorMode = getEnvAsEditorMode("EDITOR_MODE", "off")
	})
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsEditorMode(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		v := strings.ToLower(strings.TrimSpace(value))
		switch v {
		case "off", "readonly", "full":
			return v
		default:
			boolValue, err := strconv.ParseBool(value)
			if err == nil {
				if boolValue {
					return "full"
				}
				return "off"
			}
			log.Printf("Warning: unrecognized EDITOR_MODE value %q; using %q", value, fallback)
			return fallback
		}
	}
	return fallback
}

func getEnvAsPath(key string, fallback string) string {
	path := getEnv(key, fallback)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatalf("Path %s does not exist", path)
	}

	return path
}

func GetPort() string {
	return port
}

func GetDBFilePath() string {
	return dbFilePath
}

func GetDefaultChannelsPath() string {
	return defaultChannelsPath
}

func GetEditorMode() string {
	return editorMode
}
