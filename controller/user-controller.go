package controller

//go:generate mockgen -build_flags=-mod=mod -destination ../mocks/user-controller_mocks.go -package=mocks github.com/electrofelix/gin-demo/controller UserService

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/electrofelix/gin-demo/entity"
	"github.com/electrofelix/gin-demo/service"
)

type UserService interface {
	Create(ctx context.Context, user entity.User) (entity.User, error)
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
	router.POST("/users", uc.create)
	router.DELETE("/users/:email", uc.delete)
}

func (uc *UserController) create(ctx *gin.Context) {
	// should consider separate objects for internal vs external representations
	var user entity.User
	ctx.BindJSON(&user)

	password, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.MinCost)
	if err != nil {
		uc.logger.Errorf("failed to encrypted password text for new user: %s\n", user.Email)
		ctx.AbortWithStatusJSON(400, gin.H{"error": "unable to encrypt password"})

		return
	}

	// only store the encrypted password
	user.Password = string(password)
	user.LastLogin = 0

	user, err = uc.service.Create(ctx, user)
	if err != nil {
		// missing a check for already exists here
		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	user.Password = ""

	ctx.JSON(201, user)
}

func (uc *UserController) delete(ctx *gin.Context) {
	email := ctx.Param("email")

	user, err := uc.service.Delete(ctx, email)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			ctx.AbortWithStatusJSON(404, err)

			return
		}

		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	ctx.JSON(204, user)
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
