package main

import (
	"flag"
	"github.com/denismitr/goenv"
	"github.com/denismitr/resizer/cmd/internal/initialize"
	"github.com/denismitr/resizer/internal/backoffice"
	"github.com/denismitr/resizer/internal/manipulator"
	"github.com/labstack/echo/v4"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	migrate = flag.Bool("migrate", false, "Run the migrations?")
	wait = flag.Int("wait", 0, "Run the migrations?")
)

func main() {
	flag.Parse()

	if *wait != 0 {
		time.Sleep(time.Duration(*wait) * time.Second)
	}

	initialize.DotEnv()

	registry, closeRegistry := initialize.MongoRegistry(10 * time.Second, *migrate)
	defer closeRegistry()

	storage := initialize.S3StorageFromEnv()

	images := backoffice.NewImageService(registry, storage, manipulator.New(&manipulator.Config{
		AllowUpscale:        false,
		DisableOpacity:      goenv.IsTruthy("DISABLE_OPACITY"),
		SizeDiscreteStep:    goenv.IntOrDefault("DISCRETE_SIZE_STEP", 5),
		QualityDiscreteStep: goenv.IntOrDefault("DISCRETE_QUALITY_STEP", 5),
		ScaleDiscreteStep:   goenv.IntOrDefault("DISCRETE_SCALE_STEP", 5),
	}))

	server := backoffice.NewServer(echo.New(), goenv.MustString("BACKOFFICE_PORT"), images)

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGTERM, syscall.SIGINT)

	if err := server.Run(stopCh, 10 * time.Second); err != nil {
		panic(err)
	}
}
