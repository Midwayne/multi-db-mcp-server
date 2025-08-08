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
	viper.SetDefault("GOPROXY", "https://proxy.golang.org,direct")
	viper.SetDefault("TOOLS", "connect,find,aggregate,count,list-databases,list-collections,collection-indexes,collection-schema,collection-storage-size,db-stats")

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

	// Load MongoDB connection strings from environment variables
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "MONGO_CONN") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				config.MongoURIs = append(config.MongoURIs, parts[1])
			}
		}
	}

	if len(config.MongoURIs) == 0 {
		return nil, ErrNoMongoURIs
	}

	return &config, nil
}
