package backoffice

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"strings"
)

type Server struct {
	e      *echo.Echo
	images Images
}

func NewServer(e *echo.Echo, images Images) *Server {
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.DefaultCORSConfig))

	s := &Server{e: e, images: images}

	e.GET("/api/v1/images/:id", s.getImage)

	return s
}

func (s *Server) getImage(ctx echo.Context) error {
	return nil
}

func (s *Server) createNewImage(rCtx echo.Context) error {
	bucket := rCtx.FormValue("bucket")

	// Source
	file, err := rCtx.FormFile("file")
	if err != nil {
		return err // fixme
	}

	source, err := file.Open()
	if err != nil {
		return err // fixme
	}
	defer source.Close()

	useCase := createNewImage{
		originalName: file.Filename,
		originalSize: file.Size,
		originalExt:  extractExtension(file.Filename),
		bucket:       bucket,
		source:       source,

	}

	img, err := s.images.createNewImage(useCase)
	if err != nil {
		return err // fixme
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
