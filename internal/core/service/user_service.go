package service

import (
    "errors"
    "tracking/internal/core/model"
    "tracking/internal/core/repository"
)

type UserService interface {
    CreateUser(email, password, name string) (*model.User, error)
    UpdateUser(user *model.User) error
    DeleteUser(id string) error
    GetUser(id string) (*model.User, error)
    AuthenticateUser(email, password string) (*model.User, error)
}

type userService struct {
    userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
    return &userService{
        userRepo: userRepo,
    }
}

func (s *userService) CreateUser(email, password, name string) (*model.User, error) {
    if email == "" || password == "" {
        return nil, errors.New("invalid user data")
    }

    existingUser, _ := s.userRepo.FindByEmail(email)
    if existingUser != nil {
        return nil, errors.New("email already exists")
    }

    user := model.NewUser(email, password, name)
    err := s.userRepo.Create(user)
    if err != nil {
        return nil, err
    }
    return user, nil
}

func (s *userService) UpdateUser(user *model.User) error {
    if user.ID == "" {
        return errors.New("invalid user ID")
    }
    return s.userRepo.Update(user)
}

func (s *userService) DeleteUser(id string) error {
    if id == "" {
        return errors.New("invalid user ID")
    }
    return s.userRepo.Delete(id)
}

func (s *userService) GetUser(id string) (*model.User, error) {
    if id == "" {
        return nil, errors.New("invalid user ID")
    }
    return s.userRepo.FindByID(id)
}

func (s *userService) AuthenticateUser(email, password string) (*model.User, error) {
    if email == "" || password == "" {
        return nil, errors.New("invalid credentials")
    }

    user, err := s.userRepo.FindByEmail(email)
    if err != nil {
        return nil, err
    }
    if user == nil {
        return nil, errors.New("user not found")
    }

    // In production, use proper password hashing and comparison
    if user.Password != password {
        return nil, errors.New("invalid credentials")
    }

    return user, nil
}
