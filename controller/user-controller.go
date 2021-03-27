package controller

//go:generate mockgen -build_flags=-mod=mod -destination ../mocks/user-controller_mocks.go -package=mocks github.com/electrofelix/gin-demo/controller UserService

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/electrofelix/gin-demo/entity"
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
	router.GET("/users/:email", uc.get)
	router.POST("/users", uc.create)
	router.DELETE("/users/:email", uc.delete)
	router.PUT("/users/:email", uc.update)
	router.POST("/login", uc.login)
}

func (uc *UserController) create(ctx *gin.Context) {
	// should consider separate objects for internal vs external representations
	var user entity.User
	err := ctx.BindJSON(&user)
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": err.Error()})

		return
	}

	password, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.MinCost)
	if err != nil {
		uc.logger.Errorf("failed to encrypted password text for new user: %s\n", user.Email)
		ctx.AbortWithStatusJSON(400, gin.H{"error": "unable to encrypt password"})

		return
	}

	// only store the encrypted password
	user.Password = string(password)

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
		if errors.Is(err, entity.ErrNotFound) {
			ctx.AbortWithStatusJSON(404, err)

			return
		}

		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	ctx.JSON(204, user)
}


func (uc *UserController) get(ctx *gin.Context) {
	email := ctx.Param("email")

	user, err := uc.service.Get(ctx, email)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			ctx.AbortWithStatusJSON(404, err)

			return
		}

		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	// still using this terrible hack
	user.Password = ""

	ctx.JSON(200, user)
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

func (uc *UserController) login(ctx *gin.Context) {
	var credentials entity.UserLogin
	err := ctx.BindJSON(&credentials)
	if err != nil {
		ctx.AbortWithStatusJSON(400, gin.H{"error": err.Error()})

		return
	}

	user, err := uc.service.Get(ctx, credentials.Email)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			ctx.AbortWithStatusJSON(401, gin.H{"error": "Invalid Email or Password"})

			return
		}

		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	// should move this to a receiver function on the User struct
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password))
	if err != nil {
		ctx.AbortWithStatusJSON(401, gin.H{"error": "Invalid Email or Password"})

		return
	}

	// need to update last login and save
	user.LastLogin = time.Now()

	user, err = uc.service.Put(ctx, user)
	if err != nil {
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


	user, err := uc.service.Get(ctx, email)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			ctx.AbortWithStatusJSON(404, err)

			return
		}

		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	if userUpdate.Email != "" && userUpdate.Email != email {
		uc.logger.Errorf("attempted to modify email of %s to %s\n", email, userUpdate.Email)
		ctx.AbortWithStatusJSON(405, gin.H{"error": "not allowed alter email"})

		return
	}

	password, err := bcrypt.GenerateFromPassword([]byte(userUpdate.Password), bcrypt.MinCost)
	if err != nil {
		uc.logger.Errorf("failed to encrypted password text for new user: %s\n", userUpdate.Email)
		ctx.AbortWithStatusJSON(400, gin.H{"error": "unable to encrypt password"})

		return
	}

	// only store the encrypted password
	user.Password = string(password)

	// should really have separate structs for requests with pointers for field values to ensure
	// possible to determine when a specific field should be ignored as opposed to explicit request
	// to unset
	if userUpdate.Name != "" {
		user.Name = userUpdate.Name
	}

	user, err = uc.service.Put(ctx, userUpdate)
	if err != nil {
		ctx.AbortWithStatusJSON(500, gin.H{"error": "Internal Error"})

		return
	}

	user.Password = ""

	ctx.JSON(200, user)
}
