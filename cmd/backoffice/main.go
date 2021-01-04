package main

import (
	"context"
	"github.com/denismitr/goenv"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"os/signal"
	"resizer/backoffice"
	"resizer/manipulator"
	"resizer/media"
	"resizer/registry/mgoregistry"
	"resizer/storage/s3storage"
	"syscall"
	"time"
)

func main() {
	LoadFromDotEnv()

	registry, closeRegistry := InitializeMongoRegistry(30)
	defer closeRegistry()

	storage := InitializeS3StorageFromEnv()

	images := backoffice.NewImages(registry, storage, manipulator.New(true), media.NewParser())
	server := backoffice.NewServer(echo.New(), goenv.MustString("BACKOFFICE_PORT"), images)

	stopCh := make(chan os.Signal)
	signal.Notify(stopCh, syscall.SIGTERM, syscall.SIGINT)

	if err := server.Run(stopCh, 10 * time.Second); err != nil {
		panic(err)
	}
}

func InitializeS3StorageFromEnv() *s3storage.RemoteStorage {
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

func InitializeMongoRegistry(connectionTimeout time.Duration) (*mgoregistry.MongoRegistry, func()) {
	ctx, cancel := context.WithTimeout(context.Background(),connectionTimeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(goenv.MustString("MONGODB_URL")))

	if err != nil {
		panic(err)
	}

	registry := mgoregistry.New(client, mgoregistry.Config{
		DB: goenv.MustString("MONGODB_DATABASE"),
		ImagesCollection: "images", // fixme
	})

	return registry, func() {
		if err := client.Disconnect(context.Background()); err != nil {
			panic(err)
		}
	}
}

func LoadFromDotEnv(files ...string) {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}
}
