package models

import (
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type DBConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Options  map[string]any
}

type DBConnection struct {
	// Config         DBConfig
	ConnString string
	Client     *mongo.Client
}

type DBConnections map[string]*DBConnection // Key is the host name
