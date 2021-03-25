package controller

//go:generate mockgen -build_flags=-mod=mod -destination ../mocks/user-controller_mocks.go -package=mocks github.com/electrofelix/gin-demo/controller UserService

import (
	"context"

	"github.com/gin-gonic/gin"
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
	logger  *logrus.Logger
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

func WithLogger(l *logrus.Logger) Option {
	return func(uc *UserController) {
		uc.logger = l
	}
}

func (uc *UserController) RegisterRoutes(router *gin.Engine) {
	uc.logger.Info("UserController registering routes")

	router.GET("/users", uc.list)
}

func (uc *UserController) list(ctx *gin.Context) {
	users, err := uc.service.List(ctx)
	if err != nil {
		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	// ideally return different structures or implement the dynamodb marshal/unmarshal
	// interface and make password private so that it's not returned by default
	for idx := 0; idx < len(users); idx++ {
		users[idx].Password = ""
	}

	ctx.JSON(200, users)
}
