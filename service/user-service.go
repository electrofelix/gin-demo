package service

//go:generate mockgen -build_flags=-mod=mod -destination ../mocks/user-service_mocks.go -package=mocks github.com/electrofelix/gin-demo/service UserStore

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/electrofelix/gin-demo/entity"
)

type UserStore interface {
	Create(context.Context, *entity.User) error
	Delete(context.Context, string) error
	Get(context.Context, string) (*entity.User, error)
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
	if user.Email == "" {
		return entity.User{}, entity.ErrIDMissing
	}

	// move password encryption here along with any additional checks

	err := us.store.Create(ctx, &user)
	if err != nil {
		return entity.User{}, err
	}

	return user, nil
}

func (us *UserService) Delete(ctx context.Context, email string) (entity.User, error) {
	item, err := us.Get(ctx, email)
	if err != nil {
		us.logger.Errorf("error retrieving item before delete: %v", err)

		return entity.User{}, err
	}

	// Should exist at this point, thought something else may have deleted
	// it. For audit purposes would be better to mark deleted initially before
	// removal, or scrub the password in case it's still referenced by something
	// else
	err = us.store.Delete(ctx, email)
	if err != nil {
		return entity.User{}, err
	}

	return item, nil
}

func (us *UserService) Get(ctx context.Context, email string) (entity.User, error) {
	if email == "" {
		return entity.User{}, entity.ErrIDMissing
	}

	user, err := us.store.Get(ctx, email)
	if err != nil {
		return entity.User{}, err
	}

	return *user, nil
}

func (us *UserService) List(ctx context.Context) ([]entity.User, error) {
	users, err := us.store.List(ctx)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (us *UserService) Put(ctx context.Context, user entity.User) (entity.User, error) {
	if user.Email == "" {
		return entity.User{}, entity.ErrIDMissing
	}

	// move password encryption here along with any additional checks

	err := us.store.Put(ctx, &user)
	if err != nil {
		return entity.User{}, err
	}

	return user, nil
}
