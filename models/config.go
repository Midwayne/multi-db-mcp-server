package models

type Config struct {
	GoProxy   string   `mapstructure:"GOPROXY"`
	Tools     []string `mapstructure:"TOOLS"`
	MongoURIs []string `mapstructure:"MONGO_URIs"`
}
