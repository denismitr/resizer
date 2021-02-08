package backoffice

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"os"
	"resizer/media"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	port   string
	e      *echo.Echo
	images *ImageService
}

func NewServer(e *echo.Echo, port string, images *ImageService) *Server {
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.DefaultCORSConfig))

	s := &Server{e: e, images: images, port: port}

	e.GET("/api/v1/images", s.getImages)
	e.GET("/api/v1/images/:id", s.getImage)
	e.POST("/api/v1/images", s.createNewImageHandler)
	e.DELETE("/api/v1/images/:id", s.removeImage)

	return s
}

func (s *Server) getImage(rCtx echo.Context) error {
	id := rCtx.Param("id")
	if id == "" {
		return rCtx.JSON(400, map[string]string{"message": "id must be provided"})
	}

	img, err := s.images.getImage(id)
	if err != nil {
		if errors.Is(err, ErrResourceNotFound) {
			return rCtx.JSON(404, map[string]string{"message": err.Error()})
		}

		return rCtx.JSON(500, map[string]string{"message": err.Error()})
	}

	return rCtx.JSON(200, map[string]interface{}{"data": img})
}

func (s *Server) createNewImageHandler(rCtx echo.Context) error {
	namespace := rCtx.FormValue("namespace")
	if len(namespace) < 2 {
		return rCtx.JSON(400, map[string]string{"message": "namespace must be at least 1 character long"})
	}

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

	useCase := &createImageDTO{
		name:         name, // fixme: slugify original if not provided
		publish:      isTruthy(publish),
		originalName: file.Filename,
		originalSize: file.Size,
		originalExt:  extractExtension(file.Filename),
		namespace:    namespace,
		source:       source,
	}

	img, err := s.images.createNewImage(useCase)
	if err != nil {
		return rCtx.JSON(500, map[string]string{"message": err.Error()})
	}

	return rCtx.JSON(201, map[string]interface{}{"data": img})
}

func (s *Server) removeImage(rCtx echo.Context) error {
	id := rCtx.Param("id")
	if id == "" {
		return rCtx.JSON(400, map[string]string{"message": "id must be provided"})
	}

	if err := s.images.removeImage(id); err != nil {
		if errors.Is(err, ErrResourceNotFound) {
			return rCtx.JSON(404, map[string]string{"message": err.Error()})
		}

		return rCtx.JSON(500, map[string]string{"message": err.Error()})
	}

	return rCtx.JSON(204, nil)
}

func (s *Server) getImages(rCtx echo.Context) error {
	var filter media.ImageFilter

	namespace := rCtx.FormValue("namespace")
	if namespace != "" {
		filter.Namespace = namespace
	}

	page := rCtx.FormValue("page")
	if page != "" {
		v, err := strconv.Atoi(page)
		if err != nil {
			return rCtx.JSON(400, map[string]string{"message": "invalid page value " + page})
		}

		if v < 1 {
			return rCtx.JSON(400, map[string]string{"message": "invalid page value " + page})
		}

		filter.Page = uint(v)
	} else {
		filter.Page = 1
	}

	perPage := rCtx.FormValue("perPage")
	if page != "" {
		v, err := strconv.Atoi(perPage)
		if err != nil {
			return rCtx.JSON(400, map[string]string{"message": "invalid perPage value " + page})
		}

		if v < 1 {
			return rCtx.JSON(400, map[string]string{"message": "invalid perPage value " + page})
		}

		filter.PerPage = uint(v)
	} else {
		filter.PerPage = media.DefaultPerPage
	}

	collection, err := s.images.getImages(filter)
	if err != nil {
		return rCtx.JSON(500, map[string]string{"message": err.Error()})
	}

	return rCtx.JSON(200, collection)
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