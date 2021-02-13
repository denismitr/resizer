package proxy

import (
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

type Server struct {
	cfg        Config
	logger     *logrus.Logger
	httpServer *http.Server
	mux        *httpMux
}

func NewServer(cfg Config, logger *logrus.Logger, proxy ImageProxy) *Server {
	mux := newMux(cfg, proxy, logger)

	return &Server{
		cfg:    cfg,
		logger: logger,
		mux:    mux,
		httpServer: &http.Server{
			Addr:              cfg.Port,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			Handler:           mux,
			ReadHeaderTimeout: 2 * time.Second,
		},
	}
}

// Run the server
func (s *Server) Run(stopCh <-chan os.Signal, shutDownTime time.Duration) error {
	s.logger.Println("Proxy server : Starting")

	serverError := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil {
			serverError <- errors.Wrap(err, "http server error")
		}
	}()

	s.logger.Println("Proxy server : Started")

	select {
	case err := <-serverError:
		return err
	case <-stopCh:
		s.logger.Println("Proxy server : Received stop signal`")

		ctx, cancel := context.WithTimeout(context.Background(), shutDownTime)
		defer cancel()

		s.mux.stop()
		if stopErr := s.httpServer.Shutdown(ctx); stopErr != nil {
			closeErr := s.httpServer.Close()
			return errors.Wrap(closeErr, stopErr.Error())
		}

		return nil
	}
}
