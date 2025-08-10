package config

import (
	"errors"
	"os"
	"strings"

	"mongomcp/models"

	"github.com/spf13/viper"
)

var (
	ErrNoMongoURIs = errors.New("no MongoDB connection URIs found in environment variables")
)

func LoadConfig() (*models.Config, error) {
	viper.SetDefault("TOOLS", "connect,find,aggregate,count,list-databases,list-collections,collection-indexes,collection-schema,collection-storage-size,db-stats")
	viper.SetDefault("SERVE_MODE", "stdio") // Default to stdio
	viper.SetDefault("PORT", "8080")        // Default to port 8080

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Support for .env file
	if _, err := os.Stat(".env"); err == nil {
		viper.SetConfigName(".env")
		viper.SetConfigType("env")
		viper.AddConfigPath(".")
		err := viper.ReadInConfig()
		if err != nil {
			return nil, err
		}
	}

	var config models.Config
	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	for key, value := range viper.AllSettings() {
		if strings.HasPrefix(strings.ToUpper(key), "MONGO_CONN") {
			if uri, ok := value.(string); ok {
				config.MongoURIs = append(config.MongoURIs, uri)
			}
		}
	}

	if len(config.MongoURIs) == 0 {
		return nil, ErrNoMongoURIs
	}

	return &config, nil
}
