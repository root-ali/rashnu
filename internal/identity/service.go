package identity

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Repository interface {
	CreateUser(ctx context.Context, user User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUsers(ctx context.Context) ([]*User, error)
	UpdateUser(ctx context.Context, user *User) error
	UpdateUserPassword(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error
}

type Service interface {
	Create(ctx context.Context, user CreateUserRequest) error
	CreateAdminUser() error

	Update(ctx context.Context, u UpdateUserRequest) error
	UpdatePassword(ctx context.Context, upr UpdateUserPasswordRequest) error

	GetAll(ctx context.Context) ([]*User, error)

	Delete(ctx context.Context, id string) error

	Login(ctx context.Context, request LoginRequest) (LoginResponse, error)
	Logout(ctx context.Context, email string) error
	IsTokenValid(email, token string) bool
}

type service struct {
	repo         Repository
	jwtSecretKey []byte
	logger       *zap.Logger
	store        *TokenStore
}

func NewService(repo Repository, jwtSecretKey []byte, logger *zap.Logger) Service {
	return &service{repo: repo, jwtSecretKey: jwtSecretKey, logger: logger, store: NewTokenStore()}
}

func (s *service) GetAll(ctx context.Context) ([]*User, error) {
	s.logger.Debug("About to get all users")
	return s.repo.GetUsers(ctx)
}

func (s *service) Create(ctx context.Context, u CreateUserRequest) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	hashedPassword, err := hashPassword(u.Password)
	if err != nil {
		return err
	}
	user := User{
		ID:       id,
		FullName: u.FullName,
		Email:    u.Email,
		Role:     u.Role,
		Password: hashedPassword,
		Status:   u.Status,
		CreateAt: time.Now(),
		UpdateAt: time.Now(),
	}
	s.logger.Debug("creating user", zap.Any("email", user.Email), zap.Any("full_name", user.FullName),
		zap.Any("id", user.ID), zap.Any("role", user.Role), zap.Any("status", user.Status))
	return s.repo.CreateUser(ctx, user)
}

func (s *service) CreateAdminUser() error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	password, err := generateRandomPassword(20)
	if err != nil {
		return err
	}
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return err
	}
	user := User{
		ID:       id,
		FullName: "Admin",
		Email:    "default@rashnu.com",
		Password: hashedPassword,
		Role:     "admin",
		Status:   "active",
		CreateAt: time.Now(),
		UpdateAt: time.Now(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err = s.repo.CreateUser(ctx, user)
	if err == nil {
		s.logger.Info("Successfully created default admin user, please change password as soon as you can.",
			zap.String("email", user.Email), zap.String("password", password))
	}
	return err
}

func (s *service) Update(ctx context.Context, u UpdateUserRequest) error {
	user := User{
		ID:       u.ID,
		FullName: u.FullName,
		Email:    u.Email,
		Role:     u.Role,
		Status:   u.Status,
		UpdateAt: time.Now(),
	}
	s.logger.Debug("updating user", zap.Any("email", user.Email), zap.Any("full_name", user.FullName),
		zap.Any("role", user.Role), zap.Any("status", user.Status))
	err := s.repo.UpdateUser(ctx, &user)
	if err != nil {
		s.logger.Error("Error updating user",
			zap.Any("email", user.Email), zap.Any("full_name", user.FullName))
		return err
	}
	s.logger.Info("Update User successfully",
		zap.Any("email", user.Email), zap.Any("full_name", user.FullName))
	return nil
}

func (s *service) UpdatePassword(ctx context.Context, upr UpdateUserPasswordRequest) error {
	s.logger.Debug("About to update user password", zap.String("email", upr.Email))
	user, err := s.repo.GetUserByEmail(ctx, upr.Email)
	if user == nil {
		s.logger.Error("user not found", zap.String("email", upr.Email))
		return ErrUserNotFound
	}
	if err != nil {
		s.logger.Error("Error updating password", zap.String("email", upr.Email))
		return err
	}
	if user.Status != "active" {
		return ErrUserNotFound
	}
	if err := checkPassword(user.Password, upr.OldPasswrd); err != nil {
		s.logger.Error("Error updating password: old password mismatch", zap.String("email", upr.Email))
		return ErrPasswordMismatch
	}
	if upr.NewPasswrd != upr.ConfirmPassword {
		s.logger.Error("Error updating password: new password and confirm password do not match",
			zap.String("email", upr.Email))
		return ErrPasswordMismatch
	}
	if err := checkPassword(user.Password, upr.NewPasswrd); err == nil {
		s.logger.Error("Error updating password: new password match with old password", zap.String("email", upr.Email))
		return ErrPasswordMismatch
	}
	user.Password, err = hashPassword(upr.NewPasswrd)
	if err != nil {
		s.logger.Error("Error updating password: new password", zap.String("email", upr.Email))
		return err
	}
	err = s.repo.UpdateUserPassword(ctx, user)
	if err != nil {
		s.logger.Error("Error updating password", zap.String("email", upr.Email))
		return err
	}
	s.store.Delete(upr.Email)
	return nil
}

func (s *service) Logout(_ context.Context, email string) error {
	s.store.Delete(email)
	return nil
}

func (s *service) IsTokenValid(email, token string) bool {
	return s.store.IsValid(email, token)
}

func (s *service) Delete(ctx context.Context, id string) error {
	s.logger.Debug("About to delete user", zap.Any("email", id))
	err := s.repo.DeleteUser(ctx, id)
	if err != nil {
		s.logger.Error("Error deleting user", zap.Any("email", id))
		return err
	}
	s.logger.Info("User deleted", zap.Any("email", id))
	return nil
}

func (s *service) Login(ctx context.Context, request LoginRequest) (LoginResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, request.Email)
	if err != nil {
		return LoginResponse{}, err
	}
	if err := checkPassword(user.Password, request.Password); err != nil {
		return LoginResponse{}, ErrPasswordMismatch
	}
	if user.Status != "active" {
		return LoginResponse{}, ErrUserNotFound
	}

	role := user.Role
	s.logger.Info("role information", zap.String("role", role))

	expirationTime := time.Now().Add(72 * time.Hour)
	claims := &jwt.MapClaims{
		"email":     user.Email,
		"role":      role,
		"full_name": user.FullName,
		"exp":       expirationTime.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecretKey)
	if err != nil {
		return LoginResponse{}, err
	}
	s.store.Set(user.Email, tokenString)
	s.logger.Info("user logged in", zap.String("email", user.Email), zap.String("role", role))
	return LoginResponse{tokenString, user.Role, user.Email, user.FullName}, nil
}
