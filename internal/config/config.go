package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI        string
	MongoDB         string
	MongoCollection string
	Port            string
	APIKey          string
	ScrapersKey     string
	CorsOrigin      string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil {
		return Config{}, fmt.Errorf("could not load .env: %w", err)
	}
	port := os.Getenv("API_PORT")
	if _, err := strconv.Atoi(port); err != nil || port == "" {
		port = "8181"
	}
	return Config{
		MongoURI:        getenv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDB:         getenv("MONGODB_DB_NAME", "corridas_db"),
		MongoCollection: getenv("MONGODB_COLLECTION", "eventos"),
		Port:            port,
		APIKey:          os.Getenv("API_KEY"),
		ScrapersKey:     os.Getenv("SCRAPERS_API_KEY"),
		CorsOrigin:      getenv("CORS_ORIGINS", "*"),
	}, nil
}
