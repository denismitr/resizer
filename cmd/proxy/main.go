package main

import (
	"os"
	"os/signal"
	"resizer/cmd/initialize"
	"resizer/manipulator"
	"resizer/media"
	"resizer/proxy"
	"syscall"
	"time"
)

func main() {
	initialize.DotEnv()

	registry, closeRegistry := initialize.MongoRegistryFromEnv(30)
	defer closeRegistry()

	storage := initialize.S3StorageFromEnv()
	m := manipulator.New(false)

	imageProxy := proxy.NewOnTheFlyPersistingImageProxy(registry, storage, m, media.NewParser())
	server := proxy.NewServer(proxy.Config{Port: ":3333"}, imageProxy)

	stopCh := make(chan os.Signal)
	signal.Notify(stopCh, syscall.SIGTERM, syscall.SIGINT)

	if err := server.Run(stopCh, 10 * time.Second); err != nil {
		panic(err)
	}
}
