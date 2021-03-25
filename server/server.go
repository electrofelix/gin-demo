package server

//go:generate mockgen -build_flags=-mod=mod -destination ../mocks/server_mocks.go -package=mocks github.com/electrofelix/gin-demo/server Controller

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Controller interface {
	List(*gin.Context)
}

type Server struct {
	controller Controller
	address    string
	logger     *logrus.Logger
}

type Option func(*Server)

func New(ctrl Controller, options ...Option) *Server {
	s := Server{
		controller: ctrl,
		address:    ":8080",
		logger:     logrus.StandardLogger(),
	}

	for _, opt := range options {
		opt(&s)
	}

	return &s
}

func WithLogger(l *logrus.Logger) Option {
	return func(s *Server) {
		s.logger = l
	}
}

func WithAddress(addr string) Option {
	return func(s *Server) {
		s.address = addr
	}
}

func (s *Server) Start(ctx context.Context) error {
	router := gin.Default()

	srv := &http.Server{
		Addr:    s.address,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			s.logger.Fatalf("failed to server: %s\n", err)
		}
	}()

	<-ctx.Done()

	s.logger.Infoln("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Errorf("Server force to shutdown due to timeout exceeded: %v", err)

		return err
	}

	s.logger.Infoln("Shutdown complete")

	return nil
}
