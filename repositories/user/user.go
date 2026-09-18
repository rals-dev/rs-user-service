package repositories

import (
	"context"
	"errors"
	"strings"
	errWrap "user-service/common/error"
	errConstant "user-service/constants/error"
	"user-service/domain/dto"
	"user-service/domain/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

type IUserRepository interface {
	Register(context.Context, *dto.RegisterRequest) (*models.User, error)
	Update(context.Context, *dto.UpdateRequest, string) (*models.User, error)
	FindByUsername(context.Context, string) (*models.User, error)
	FindByEmail(context.Context, string) (*models.User, error)
	FindByUUID(context.Context, string) (*models.User, error)
}

func NewUserRepository(db *gorm.DB) IUserRepository {
	return &UserRepository{db: db}
}

// isUniqueConstraintViolation reports whether err is a Postgres unique-constraint
// violation on the given constraint (matched by name, e.g. one of the
// uniqueIndex names declared on models.User). String matching is used instead
// of importing the pgconn error type, since the error message Postgres/pgx
// returns always includes the constraint name.
func isUniqueConstraintViolation(err error, constraintName string) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") && strings.Contains(msg, constraintName)
}

func (r *UserRepository) Register(ctx context.Context, req *dto.RegisterRequest) (*models.User, error) {
	user := models.User{
		UUID:        uuid.New(),
		Name:        req.Name,
		Username:    req.Username,
		Password:    req.Password,
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
		RoleID:      req.RoleID,
	}

	err := r.db.WithContext(ctx).Create(&user).Error
	if err != nil {
		if isUniqueConstraintViolation(err, "idx_users_username") {
			return nil, errConstant.ErrUserAlreadyExist
		}
		if isUniqueConstraintViolation(err, "idx_users_email") {
			return nil, errConstant.ErrEmailAlreadyExist
		}
		return nil, errWrap.WrapError(errConstant.ErrSQLError)
	}
	return &user, nil
}
func (r *UserRepository) Update(ctx context.Context, req *dto.UpdateRequest, paramUUID string) (*models.User, error) {
	if req == nil || req.Password == nil {
		return nil, errConstant.ErrRequestValidation
	}

	userUUID, err := uuid.Parse(paramUUID)
	if err != nil {
		return nil, errConstant.ErrRequestValidation
	}
	user := models.User{
		UUID:        userUUID,
		Name:        req.Name,
		Username:    req.Username,
		Password:    *req.Password,
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
	}
	err = r.db.WithContext(ctx).
		Where("uuid = ?", paramUUID).Omit("uuid").
		Updates(&user).Error
	if err != nil {
		if isUniqueConstraintViolation(err, "idx_users_username") {
			return nil, errConstant.ErrUserNameExist
		}
		if isUniqueConstraintViolation(err, "idx_users_email") {
			return nil, errConstant.ErrEmailAlreadyExist
		}
		return nil, errWrap.WrapError(errConstant.ErrSQLError)
	}
	return &user, nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("Role").Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errConstant.ErrUserNotFound
		}
		return nil, errWrap.WrapError(errConstant.ErrSQLError)
	}
	return &user, nil
}
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("Role").Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errConstant.ErrUserNotFound
		}
		return nil, errWrap.WrapError(errConstant.ErrSQLError)
	}
	return &user, nil
}

func (r *UserRepository) FindByUUID(ctx context.Context, paramUUID string) (*models.User, error) {
	if _, err := uuid.Parse(paramUUID); err != nil {
		return nil, errConstant.ErrRequestValidation
	}
	var user models.User
	err := r.db.WithContext(ctx).Preload("Role").Where("uuid = ?", paramUUID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errConstant.ErrUserNotFound
		}
		return nil, errWrap.WrapError(errConstant.ErrSQLError)
	}
	return &user, nil
}
