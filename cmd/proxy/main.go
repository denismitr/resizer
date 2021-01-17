package main

import (
	"github.com/denismitr/goenv"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"resizer/cmd/initialize"
	"resizer/manipulator"
	"resizer/proxy"
	"syscall"
	"time"
)

func main() {
	initialize.DotEnv()

	registry, closeRegistry := initialize.MongoRegistry(30 * time.Second, false)
	defer closeRegistry()

	storage := initialize.S3StorageFromEnv()
	m := manipulator.New(&manipulator.Config{ // fixme: dotenv
		AllowUpscale:        false,
		DisableOpacity:      goenv.IsTruthy("DISABLE_OPACITY"),
		SizeDiscreteStep:    goenv.IntOrDefault("DISCRETE_SIZE_STEP", 5),
		QualityDiscreteStep: goenv.IntOrDefault("DISCRETE_QUALITY_STEP", 5),
		ScaleDiscreteStep:   goenv.IntOrDefault("DISCRETE_SCALE_STEP", 5),
	})

	log := logrus.New()
	log.Out = os.Stderr
	log.Formatter = &logrus.TextFormatter{
		TimestampFormat: time.StampMilli,
		FullTimestamp:   true,
	}

	imageProxy := proxy.NewOnTheFlyPersistingImageProxy(log, registry, storage, m)
	server := proxy.NewServer(proxy.Config{Port: ":3333"}, log, imageProxy)

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGTERM, syscall.SIGINT)

	if err := server.Run(stopCh, 10 * time.Second); err != nil {
		panic(err)
	}
}
