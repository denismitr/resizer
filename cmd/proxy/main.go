package main

import (
	"github.com/denismitr/goenv"
	"github.com/denismitr/resizer/cmd/internal/initialize"
	"github.com/denismitr/resizer/internal/media/manipulator"
	"github.com/denismitr/resizer/internal/proxy"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		logrus.Printf("\nRESIZER PROXY ERROR : %s", err.Error())
		os.Exit(1)
	}
}

func run() error {
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
	server := proxy.NewServer(proxy.Config{
		Port:        goenv.StringOrDefault("PROXY_PORT", ":3000"),
		ReadTimeout: goenv.DurationOrDefault("PROXY_TIMEOUT", time.Second, 2 * time.Second),
		WriteTimeout: goenv.DurationOrDefault("PROXY_TIMEOUT", time.Second, 2 * time.Second),
	}, log, imageProxy)

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGTERM, syscall.SIGINT)

	if err := server.Run(stopCh, 10 * time.Second); err != nil {
		return err
	}

	return nil
}
