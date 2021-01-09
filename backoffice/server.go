package backoffice

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"os"
	"strings"
	"time"
)

type Server struct {
	port   string
	e      *echo.Echo
	images *Images
}

func NewServer(e *echo.Echo, port string, images *Images) *Server {
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.DefaultCORSConfig))

	s := &Server{e: e, images: images, port: port}

	e.GET("/api/v1/images/:id", s.getImage)
	e.POST("/api/v1/images", s.createNewImage)

	return s
}

func (s *Server) getImage(ctx echo.Context) error {
	return nil
}

func (s *Server) createNewImage(rCtx echo.Context) error {
	bucket := rCtx.FormValue("bucket")
	name := rCtx.FormValue("name")
	publish := rCtx.FormValue("publish")

	// Source
	file, err := rCtx.FormFile("file")
	if err != nil {
		return rCtx.JSON(400, map[string]string{"message": err.Error()})
	}

	source, err := file.Open()
	if err != nil {
		return rCtx.JSON(500, map[string]string{"message": err.Error()})
	}
	defer func () {
		if err := source.Close(); err != nil {
			s.e.Logger.Error(err)
		}

		if err := os.Remove(file.Filename); err != nil {
		  	s.e.Logger.Error(err)
		}
	}()

	useCase := &createNewImage{
		name: name, // fixme: slugify original if not provided
		publish: isTruthy(publish),
		originalName: file.Filename,
		originalSize: file.Size,
		originalExt:  extractExtension(file.Filename),
		bucket:       bucket,
		source:       source,
	}

	img, err := s.images.createNewImage(useCase)
	if err != nil {
		return rCtx.JSON(500, map[string]string{"message": err.Error()})
	}

	return rCtx.JSON(201, map[string]interface{}{"data": img})
}

func (s *Server) Run(stopCh <-chan os.Signal, shutDownTime time.Duration) error {
	go func() {
		if err := s.e.Start(s.port); err != nil {
			s.e.Logger.Info("shutting down the server")
		}
	}()

	<-stopCh
	ctx, cancel := context.WithTimeout(context.Background(), shutDownTime)
	defer cancel()
	if err := s.e.Shutdown(ctx); err != nil {
		s.e.Logger.Error(err)
		return err
	}

	return nil
}

func extractExtension(filename string) string {
	segments := strings.Split(strings.Trim(filename, " "), ".")
	if len(segments) < 2 {
		return ""
	}

	return segments[len(segments)-1]
}

func isTruthy(input string) bool {
	input = strings.ToLower(input)
	if input == "on" || input == "true" || input == "1" {
		return true
	}
	return false
}