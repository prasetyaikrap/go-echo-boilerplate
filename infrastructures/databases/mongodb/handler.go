package mongodb

import (
	"context"
	"fmt"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/configurations"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// MongoInstance wraps a single MongoDB connection and database. Create one per database the service needs, e.g.
//
//	mainMongo := mongodb.NewMongoInstance(configs, configs.Envs.MainMongo, mongodb.CollectionIndexes{...})
//	auditMongo := mongodb.NewMongoInstance(configs, configs.Envs.AuditMongo)
//
// Use Collection to query with the official driver API.
type MongoInstance struct {
	Name     string
	Client   *mongo.Client
	Database *mongo.Database
}

// CollectionIndexes lists the indexes a collection needs. They are created on startup when AutoMigrate is enabled
type CollectionIndexes struct {
	Collection string
	Indexes    []mongo.IndexModel
}

func NewMongoInstance(configs *configurations.Configs, mongoConfig models.MongoConfig, collectionIndexes ...CollectionIndexes) *MongoInstance {
	if mongoConfig.Database == "" {
		configs.Logger.Fatal("MongoDB database name is not configured", "name", mongoConfig.Name)
	}

	clientOptions := options.Client().ApplyURI(mongoConfig.URI).SetAppName(configs.Envs.Application.ClientID)
	if mongoConfig.MaxPoolSize > 0 {
		clientOptions.SetMaxPoolSize(mongoConfig.MaxPoolSize)
	}
	if mongoConfig.MinPoolSize > 0 {
		clientOptions.SetMinPoolSize(mongoConfig.MinPoolSize)
	}
	if mongoConfig.MaxConnIdleTime > 0 {
		clientOptions.SetMaxConnIdleTime(mongoConfig.MaxConnIdleTime)
	}
	if mongoConfig.ConnectTimeout > 0 {
		clientOptions.SetConnectTimeout(mongoConfig.ConnectTimeout)
		clientOptions.SetServerSelectionTimeout(mongoConfig.ConnectTimeout)
	}

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		configs.Logger.Fatal("MongoDB initialization failed", "name", mongoConfig.Name, "error", err)
	}

	// Connect is lazy, so ping to fail fast when the server is unreachable or credentials are wrong
	ctx := context.Background()
	if mongoConfig.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, mongoConfig.ConnectTimeout)
		defer cancel()
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		configs.Logger.Fatal("MongoDB connection failed", "name", mongoConfig.Name, "error", err)
	}

	configs.Logger.Info("MongoDB Successfully Connected", "name", mongoConfig.Name, "database", mongoConfig.Database)

	instance := &MongoInstance{
		Name:     mongoConfig.Name,
		Client:   client,
		Database: client.Database(mongoConfig.Database),
	}

	if mongoConfig.AutoMigrate {
		if err := instance.CreateIndexes(ctx, collectionIndexes...); err != nil {
			configs.Logger.Fatal("MongoDB migration failed", "name", mongoConfig.Name, "error", err)
		}

		configs.Logger.Info("MongoDB Migration Successful", "name", mongoConfig.Name)
	}

	return instance
}

// Collection returns a handle to a collection in the configured database
func (m *MongoInstance) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

// CreateIndexes creates the given indexes. Existing indexes with the same definition are left as they are,
// but an index with the same name and a different definition returns an error
func (m *MongoInstance) CreateIndexes(ctx context.Context, collectionIndexes ...CollectionIndexes) error {
	for _, collection := range collectionIndexes {
		if len(collection.Indexes) == 0 {
			continue
		}

		if _, err := m.Collection(collection.Collection).Indexes().CreateMany(ctx, collection.Indexes); err != nil {
			return fmt.Errorf("mongodb %s create indexes on %q: %w", m.Name, collection.Collection, err)
		}
	}

	return nil
}

// Ping checks the MongoDB connection. Useful as a readiness check
func (m *MongoInstance) Ping(ctx context.Context) error {
	return m.Client.Ping(ctx, readpref.Primary())
}

// Close disconnects from MongoDB, waiting for in-progress operations until ctx is done
func (m *MongoInstance) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}
