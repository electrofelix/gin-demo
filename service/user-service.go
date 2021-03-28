package service

//go:generate mockgen -build_flags=-mod=mod -destination ../mocks/user-service_mocks.go -package=mocks github.com/electrofelix/gin-demo/service UserStore

import (
	"context"
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/electrofelix/gin-demo/entity"
)

type UserStore interface {
	Create(context.Context, *entity.User) error
	Delete(context.Context, string) error
	GetByEmail(context.Context, string) (*entity.User, error)
	GetById(context.Context, string) (*entity.User, error)
	List(context.Context) ([]entity.User, error)
	Put(context.Context, *entity.User) error
}

type UserService struct {
	store  UserStore
	logger *logrus.Logger
}

type Option func(*UserService)

func New(store UserStore, options ...Option) *UserService {
	us := &UserService{
		store:  store,
		logger: logrus.StandardLogger(),
	}

	for _, opt := range options {
		opt(us)
	}

	return us
}

func (us *UserService) Create(ctx context.Context, user entity.User) (entity.User, error) {
	// create the new user id
	user.Id = xid.New().String()

	password, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.MinCost)
	if err != nil {
		us.logger.Errorf("failed to encrypted password text for new user: %s\n", user.Email)

		return entity.User{}, entity.ErrInternalError
	}

	user.Password = string(password)

	err = us.store.Create(ctx, &user)
	if err != nil {
		return entity.User{}, err
	}

	user.Password = ""

	return user, nil
}

func (us *UserService) Delete(ctx context.Context, id string) (entity.User, error) {
	user, err := us.Get(ctx, id)
	if err != nil {
		us.logger.Errorf("error retrieving item before delete: %v", err)

		return entity.User{}, err
	}

	// Should exist at this point, thought something else may have deleted
	// it. For audit purposes would be better to mark deleted initially before
	// removal, or scrub the password in case it's still referenced by something
	// else
	err = us.store.Delete(ctx, id)
	if err != nil {
		return entity.User{}, err
	}

	return user, nil
}

func (us *UserService) Get(ctx context.Context, id string) (entity.User, error) {
	if err := validateId(id); err != nil {
		return entity.User{}, err
	}

	user, err := us.store.GetById(ctx, id)
	if err != nil {
		return entity.User{}, err
	}

	respUser := *user
	// technically the task didn't specify whether Password needs to be
	// protected, just assuming that it's probably a good idea
	// If enough time to clean up, should look at either handling this directly
	// on the entity.User or providing separate types between responses and requests
	// to customize the fields required and returned.
	respUser.Password = ""

	return respUser, nil
}

func (us *UserService) List(ctx context.Context) ([]entity.User, error) {
	users, err := us.store.List(ctx)
	if err != nil {
		return nil, err
	}

	// ideally return different structures or implement the dynamodb marshal/unmarshal
	// interface and make password private so that it's not returned by default
	for idx := 0; idx < len(users); idx++ {
		users[idx].Password = ""
	}

	return users, nil
}

func (us *UserService) Put(ctx context.Context, user entity.User) (entity.User, error) {
	if err := validateId(user.Id); err != nil {
		return entity.User{}, err
	}

	err := us.store.Put(ctx, &user)
	if err != nil {
		return entity.User{}, err
	}

	return user, nil
}

func (us *UserService) Update(ctx context.Context, id string, user entity.User) (entity.User, error) {
	if err := validateId(id); err != nil {
		return entity.User{}, err
	}

	currentUser, err := us.Get(ctx, id)
	if err != nil {
		return entity.User{}, err
	}

	if user.Password != "" {
		password, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.MinCost)
		if err != nil {
			us.logger.Errorf("failed to encrypted password text for new user: %s\n", id)

			return entity.User{}, entity.ErrInternalError
		}

		currentUser.Password = string(password)
	}

	// should really have separate structs for requests with pointers for field values to ensure
	// possible to determine when a specific field should be ignored as opposed to explicit request
	// to unset
	if user.Name != "" {
		currentUser.Name = user.Name
	}

	err = us.store.Put(ctx, &currentUser)
	if err != nil {
		us.logger.Errorf("failed to store updated user information: %s\n", currentUser.Id)
		return entity.User{}, err
	}

	currentUser.Password = ""

	return currentUser, nil
}

func (us *UserService) ValidateCredentials(ctx context.Context, credentials entity.UserLogin) error {
	user, err := us.store.GetByEmail(ctx, credentials.Email)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return entity.ErrBadCredentials
		}

		us.logger.Errorf("failed to retrieve user '%s', unexpected error: %v", credentials.Email, err)

		return entity.ErrInternalError
	}

	// should move this to a receiver function on the User struct?
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password))
	if err != nil {
		return entity.ErrBadCredentials
	}

	user.LastLogin = time.Now()

	err = us.store.Put(ctx, user)
	if err != nil {
		us.logger.Errorf("failed to update user '%s', last login time unexpected error: %v", credentials.Email, err)

		return entity.ErrInternalError
	}

	return nil
}

func validateId(id string) error {
	if id == "" {
		return entity.ErrIDMissing
	}

	if _, err := xid.FromString(id); err != nil {
		return entity.ErrIDInvalid
	}

	return nil
}
