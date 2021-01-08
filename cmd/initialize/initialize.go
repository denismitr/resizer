package initialize

import (
	"context"
	"github.com/denismitr/goenv"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"resizer/registry/mgoregistry"
	"resizer/storage/s3storage"
	"time"
)

func S3StorageFromEnv() *s3storage.RemoteStorage {
	cfg := s3storage.Config{
		AccessKey: goenv.MustString("S3_ACCESS_KEY_ID"),
		AccessSecret: goenv.MustString("S3_SECRET_ACCESS_KEY"),
		AccessToken: "",
		Region: goenv.MustString("S3_REGION"),
		Endpoint: goenv.MustString("S3_ENDPOINT"),
		S3ForcePathStyle: goenv.IsTruthy("S3_FORCE_PATH_STYLE"),
		EnableSSL: goenv.IsTruthy("S3_SSL"),
	}

	return s3storage.New(cfg)
}

func MongoRegistry(connectionTimeout time.Duration, migrate bool) (*mgoregistry.MongoRegistry, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(goenv.MustString("MONGODB_URL")))
	if err != nil {
		panic(err)
	}

	registry := mgoregistry.New(client, mgoregistry.Config{
		DB: goenv.MustString("MONGODB_DATABASE"),
		ImagesCollection: "images",
		SlicesCollection: "slices",
	})

	if migrate {
		if err := registry.Migrate(ctx); err != nil {
			panic(err)
		}
	}

	return registry, func() {
		if err := client.Disconnect(context.Background()); err != nil {
			panic(err)
		}
	}
}

func DotEnv(files ...string) {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}
}
