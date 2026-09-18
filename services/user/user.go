package services

import (
	"context"
	"errors"
	"strings"
	"time"
	"user-service/config"
	"user-service/constants"
	errConstant "user-service/constants/error"
	"user-service/domain/dto"
	"user-service/domain/models"
	"user-service/repositories"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repository repositories.IRepositoryRegistry
}

type IUserService interface {
	Login(context.Context, *dto.LoginRequest) (*dto.LoginResponse, error)
	Register(context.Context, *dto.RegisterRequest) (*dto.RegisterResponse, error)
	Update(context.Context, *dto.UpdateRequest, string) (*dto.UserResponse, error)
	GetUserLogin(context.Context) (*dto.UserResponse, error)
	GetUserByUUID(context.Context, string) (*dto.UserResponse, error)
}

type Claims struct {
	User *dto.UserResponse
	jwt.RegisteredClaims
}

func NewUserService(repository repositories.IRepositoryRegistry) IUserService {
	return &UserService{repository: repository}
}

// getUserLogin extracts the authenticated caller's claims previously attached
// to the context by middlewares.Authenticate.
func getUserLogin(ctx context.Context) (*dto.UserResponse, error) {
	userLogin, ok := ctx.Value(constants.UserLogin).(*dto.UserResponse)
	if !ok || userLogin == nil {
		return nil, errConstant.ErrUnauthorized
	}
	return userLogin, nil
}

// authorizeSelfOrAdmin enforces that a GET/PUT on a given user UUID is only
// allowed for the JWT owner (matching UUID) or a caller with the admin role.
// It also validates that targetUUID is a well-formed UUID before it ever
// reaches the database.
func authorizeSelfOrAdmin(ctx context.Context, targetUUID string) (*dto.UserResponse, error) {
	parsedUUID, err := uuid.Parse(targetUUID)
	if err != nil {
		return nil, errConstant.ErrRequestValidation
	}

	userLogin, err := getUserLogin(ctx)
	if err != nil {
		return nil, err
	}

	if strings.ToLower(userLogin.Role) == constants.AdminRoleCode {
		return userLogin, nil
	}
	if userLogin.UUID == parsedUUID {
		return userLogin, nil
	}
	return nil, errConstant.ErrForbidden
}

func (u *UserService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := u.repository.GetUser().FindByUsername(ctx, req.Username)
	if err != nil {
		// A missing username and a wrong password must produce the exact same
		// error, otherwise the response leaks whether a username is registered
		// (user enumeration).
		if errors.Is(err, errConstant.ErrUserNotFound) {
			return nil, errConstant.ErrInvalidCredentials
		}
		return nil, err
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errConstant.ErrInvalidCredentials
	}

	expTime := time.Now().Add(time.Duration(config.Config.JwtExpirationTime) * time.Minute).Unix()
	data := &dto.UserResponse{
		UUID:        user.UUID,
		Name:        user.Name,
		Username:    user.Username,
		PhoneNumber: user.PhoneNumber,
		Email:       user.Email,
		Role:        strings.ToLower(user.Role.Code),
	}

	claims := &Claims{
		User: data,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Unix(expTime, 0)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(config.Config.JwtSecretKey))

	if err != nil {
		return nil, err
	}
	response := &dto.LoginResponse{
		User:  *data,
		Token: tokenString,
	}
	return response, nil
}

// findUserByUsername returns (nil, nil) when no user has that username,
// distinguishing "not found" from a real lookup error.
func (u *UserService) findUserByUsername(ctx context.Context, username string) (*models.User, error) {
	user, err := u.repository.GetUser().FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, errConstant.ErrUserNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

// findUserByEmail returns (nil, nil) when no user has that email,
// distinguishing "not found" from a real lookup error.
func (u *UserService) findUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := u.repository.GetUser().FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, errConstant.ErrUserNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (u *UserService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	if req.Password != req.ConfirmPassword {
		return nil, errConstant.ErrPasswordDoesNotMatch
	}

	existingUsername, err := u.findUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existingUsername != nil {
		return nil, errConstant.ErrUserAlreadyExist
	}

	existingEmail, err := u.findUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existingEmail != nil {
		return nil, errConstant.ErrEmailAlreadyExist
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := u.repository.GetUser().Register(ctx, &dto.RegisterRequest{
		Name:        req.Name,
		Username:    req.Username,
		Password:    string(hashedPassword),
		Email:       req.Email,
		RoleID:      constants.Customer,
		PhoneNumber: req.PhoneNumber,
	})

	if err != nil {
		return nil, err
	}

	response := &dto.RegisterResponse{
		User: dto.UserResponse{
			UUID:        user.UUID,
			Name:        user.Name,
			Username:    user.Username,
			Email:       user.Email,
			Role:        strings.ToLower(user.Role.Code),
			PhoneNumber: user.PhoneNumber,
		},
	}
	return response, nil
}

func (u *UserService) Update(ctx context.Context, req *dto.UpdateRequest, targetUUID string) (*dto.UserResponse, error) {
	if _, err := authorizeSelfOrAdmin(ctx, targetUUID); err != nil {
		return nil, err
	}

	user, err := u.repository.GetUser().FindByUUID(ctx, targetUUID)
	if err != nil {
		return nil, err
	}

	if req.Username != user.Username {
		existing, err := u.findUserByUsername(ctx, req.Username)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, errConstant.ErrUserNameExist
		}
	}

	if req.Email != user.Email {
		existing, err := u.findUserByEmail(ctx, req.Email)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, errConstant.ErrEmailAlreadyExist
		}
	}

	password := ""
	if req.Password != nil {
		if req.ConfirmPassword == nil || *req.Password != *req.ConfirmPassword {
			return nil, errConstant.ErrPasswordDoesNotMatch
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		password = string(hashedPassword)
	}

	userResult, err := u.repository.GetUser().Update(ctx, &dto.UpdateRequest{
		Name:        req.Name,
		Username:    req.Username,
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
		Password:    &password,
	}, targetUUID)

	if err != nil {
		return nil, err
	}

	data := dto.UserResponse{
		UUID:        userResult.UUID,
		Name:        userResult.Name,
		Username:    userResult.Username,
		Email:       userResult.Email,
		PhoneNumber: userResult.PhoneNumber,
	}
	return &data, nil
}

func (u *UserService) GetUserLogin(ctx context.Context) (*dto.UserResponse, error) {
	userLogin, err := getUserLogin(ctx)
	if err != nil {
		return nil, err
	}
	data := dto.UserResponse{
		UUID:        userLogin.UUID,
		Name:        userLogin.Name,
		Username:    userLogin.Username,
		Email:       userLogin.Email,
		Role:        userLogin.Role,
		PhoneNumber: userLogin.PhoneNumber,
	}
	return &data, nil
}

func (u *UserService) GetUserByUUID(ctx context.Context, targetUUID string) (*dto.UserResponse, error) {
	if _, err := authorizeSelfOrAdmin(ctx, targetUUID); err != nil {
		return nil, err
	}

	user, err := u.repository.GetUser().FindByUUID(ctx, targetUUID)
	if err != nil {
		return nil, err
	}
	data := dto.UserResponse{
		UUID:        user.UUID,
		Name:        user.Name,
		Username:    user.Username,
		Email:       user.Email,
		Role:        strings.ToLower(user.Role.Code),
		PhoneNumber: user.PhoneNumber,
	}
	return &data, nil
}
