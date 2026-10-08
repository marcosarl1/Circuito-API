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
	AWSBucketName   string
	AWSRegion       string
	AWSAccessKeyID  string
	AWSSecretKey    string
	BucketJSONKey   string
	JWTSecret       string
	AdminEmail      string
	AdminPassword   string
	ScraperTrigger  ScraperTriggerConfig
}

// ScraperTriggerConfig controls the automatic start of the worker
// execution after POST /scrape/run. Disabled by default: with no managed
// identity (local dev, tests) the API keeps queue-only behavior.
type ScraperTriggerConfig struct {
	Enabled        bool
	SubscriptionID string
	ResourceGroup  string
	JobName        string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() (Config, error) {
	if _, statErr := os.Stat(".env"); statErr == nil {
		if err := godotenv.Load(); err != nil {
			return Config{}, fmt.Errorf("could not load .env %w", err)
		}
	} else if !os.IsNotExist(statErr) {
		return Config{}, fmt.Errorf("could not access .env: %w", statErr)
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
		AWSBucketName:   os.Getenv("AWS_BUCKET_NAME"),
		AWSRegion:       getenv("AWS_REGION", "us-east-1"),
		AWSAccessKeyID:  os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretKey:    os.Getenv("AWS_SECRET_ACCESS_KEY"),
		BucketJSONKey:   getenv("BUCKET_JSON_KEY", "eventos_real.json"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AdminEmail:      os.Getenv("ADMIN_EMAIL"),
		AdminPassword:   os.Getenv("ADMIN_PASSWORD"),
		ScraperTrigger: ScraperTriggerConfig{
			Enabled:        os.Getenv("SCRAPER_TRIGGER_ENABLED") == "true",
			SubscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
			ResourceGroup:  getenv("AZURE_RESOURCE_GROUP", "rg-circuitoapp"),
			JobName:        getenv("SCRAPER_JOB_NAME", "correpb-scraper"),
		},
	}, nil
}
