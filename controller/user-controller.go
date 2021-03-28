package controller

//go:generate mockgen -build_flags=-mod=mod -destination ../mocks/user-controller_mocks.go -package=mocks github.com/electrofelix/gin-demo/controller UserService

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/electrofelix/gin-demo/entity"
)

type UserService interface {
	Create(ctx context.Context, user entity.User) (entity.User, error)
	Delete(ctx context.Context, id string) (entity.User, error)
	Get(ctx context.Context, id string) (entity.User, error)
	List(ctx context.Context) ([]entity.User, error)
	Update(ctx context.Context, id string, user entity.User) (entity.User, error)
	ValidateCredentials(ctx context.Context, credentials entity.UserLogin) error
}

type UserController struct {
	service UserService
	logger  *logrus.Logger
}

type Option func(*UserController)

func New(service UserService, router gin.IRoutes, opts ...Option) *UserController {
	controller := &UserController{
		service: service,
		logger:  logrus.StandardLogger(),
	}

	for _, opt := range opts {
		opt(controller)
	}

	controller.logger.Info("UserController registering routes")

	router.GET("/users", controller.list)
	router.GET("/users/:email", controller.get)
	router.POST("/users", controller.create)
	router.DELETE("/users/:email", controller.delete)
	router.PATCH("/users/:email", controller.update)
	router.POST("/login", controller.login)

	return controller
}

func WithLogger(l *logrus.Logger) Option {
	return func(uc *UserController) {
		uc.logger = l
	}
}

func (uc *UserController) create(ctx *gin.Context) {
	// should consider separate objects for internal vs external representations
	var user entity.User
	err := ctx.BindJSON(&user)
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": err.Error()})

		return
	}

	userResp, err := uc.service.Create(ctx, user)
	if err != nil {
		if errors.Is(err, entity.ErrIDCollision) {
			// could potentially return 201 here as well
			ctx.AbortWithStatusJSON(409, gin.H{"error": err.Error()})

			return
		}

		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	ctx.JSON(201, userResp)
}

func (uc *UserController) delete(ctx *gin.Context) {
	email := ctx.Param("email")

	userResp, err := uc.service.Delete(ctx, email)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			ctx.AbortWithStatusJSON(404, err)

			return
		}

		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	ctx.JSON(204, userResp)
}

func (uc *UserController) get(ctx *gin.Context) {
	email := ctx.Param("email")

	userResp, err := uc.service.Get(ctx, email)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			ctx.AbortWithStatusJSON(404, err)

			return
		}

		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	ctx.JSON(200, userResp)
}

func (uc *UserController) list(ctx *gin.Context) {
	usersResp, err := uc.service.List(ctx)
	if err != nil {
		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	ctx.JSON(200, usersResp)
}

func (uc *UserController) login(ctx *gin.Context) {
	var credentials entity.UserLogin
	err := ctx.BindJSON(&credentials)
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": err.Error()})

		return
	}

	err = uc.service.ValidateCredentials(ctx, credentials)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			ctx.AbortWithStatusJSON(401, gin.H{"error": "Invalid Email or Password"})

			return
		}

		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	ctx.JSON(200, gin.H{"status": "SUCCESS"})
}

func (uc *UserController) update(ctx *gin.Context) {
	email := ctx.Param("email")

	var userUpdate entity.User
	err := ctx.BindJSON(&userUpdate)
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": err.Error()})

		return
	}


	user, err := uc.service.Update(ctx, email, userUpdate)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			ctx.AbortWithStatusJSON(404, err)

			return
		}

		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	ctx.JSON(200, user)
}
