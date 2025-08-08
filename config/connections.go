package config

import (
	"fmt"
	"mongomcp/constants"
	"mongomcp/models"
	"mongomcp/utils"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// local type alias for models.DBConnections
type DBConnections models.DBConnections

func NewConnectionManager(appConfig models.Config) (*DBConnections, error) {
	connections := make(models.DBConnections)

	for _, uri := range appConfig.MongoURIs {
		host, err := utils.ExtractMongoHost(uri)
		if err != nil {
			// fmt.Printf("Invalid Mongo URI %s: %v\n", uri, err)
			continue
		}

		clientOpts := options.Client().
			ApplyURI(uri).
			SetMaxConnIdleTime(constants.InactivityTimeout)

		client, err := mongo.Connect(clientOpts)
		if err != nil {
			fmt.Printf("Failed to connect to MongoDB host %s: %v\n", host, err)
			continue
		}

		connections[host] = &models.DBConnection{
			ConnString: uri,
			Client:     client,
		}
	}

	if len(connections) == 0 {
		fmt.Println("No valid MongoDB connections established")
		return nil, fmt.Errorf("no valid MongoDB connections established")
	}

	dbConns := DBConnections(connections)
	return &dbConns, nil
}

func (connections *DBConnections) GetConnection(host string) (*mongo.Client, error) {
	if conn, exists := (*connections)[host]; exists {
		return conn.Client, nil
	}
	return nil, fmt.Errorf("no connection found for host: %s", host)
}
