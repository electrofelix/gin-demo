package controller

//go:generate mockgen -build_flags=-mod=mod -destination ../mocks/user-controller_mocks.go -package=mocks github.com/electrofelix/gin-demo/controller UserService

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/electrofelix/gin-demo/entity"
)

type UserService interface {
	Delete(ctx context.Context, id string) (entity.User, error)
	Get(ctx context.Context, id string) (entity.User, error)
	List(ctx context.Context) ([]entity.User, error)
	Put(ctx context.Context, user entity.User) (entity.User, error)
}

type UserController struct {
	service UserService
	logger  logrus.StdLogger
}

type Option func(*UserController)

func New(service UserService, opts ...Option) *UserController {
	controller := &UserController{
		service: service,
		logger:  logrus.StandardLogger(),
	}

	for _, opt := range opts {
		opt(controller)
	}

	return controller
}

func WithLogger(l logrus.StdLogger) Option {
	return func(uc *UserController) {
		uc.logger = l
	}
}
