package models

type Config struct {
	Tools     []string `mapstructure:"TOOLS"`
	MongoURIs []string `mapstructure:"MONGO_URIs"`
	ServeMode string   `mapstructure:"SERVE_MODE"`
	Port      string   `mapstructure:"PORT"`
}
