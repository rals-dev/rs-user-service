package seeders

import (
	"os"
	"strings"
	"user-service/config"
	"user-service/constants"
	"user-service/domain/models"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// minAdminPasswordLength mirrors the minimum password length enforced on the
// public register/update endpoints (see domain/dto/user.go validate tags).
const minAdminPasswordLength = 8

// RunUserSeeder seeds an initial admin user, but only from explicit,
// validated environment variables. There is no hardcoded default
// username/password: if ADMIN_USERNAME, ADMIN_PASSWORD, ADMIN_EMAIL and
// ADMIN_PHONE_NUMBER are not all set, seeding is skipped entirely rather than
// falling back to a guessable default credential (which would be especially
// dangerous if that fallback ever ran in production).
func RunUserSeeder(db *gorm.DB) {
	username := strings.TrimSpace(os.Getenv("ADMIN_USERNAME"))
	password := os.Getenv("ADMIN_PASSWORD")
	email := strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	phoneNumber := strings.TrimSpace(os.Getenv("ADMIN_PHONE_NUMBER"))

	if username == "" || password == "" || email == "" || phoneNumber == "" {
		if config.Config.AppEnv == config.ProductionEnv {
			logrus.Warn("admin seeder skipped: ADMIN_USERNAME, ADMIN_PASSWORD, ADMIN_EMAIL and ADMIN_PHONE_NUMBER must all be set to seed an admin user in production; no default credential will be created")
			return
		}
		logrus.Warn("admin seeder skipped: set ADMIN_USERNAME, ADMIN_PASSWORD, ADMIN_EMAIL and ADMIN_PHONE_NUMBER to seed a development admin user")
		return
	}

	if len(password) < minAdminPasswordLength {
		logrus.Errorf("admin seeder skipped: ADMIN_PASSWORD must be at least %d characters", minAdminPasswordLength)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logrus.Errorf("failed to hash admin password: %v", err)
		panic(err)
	}

	user := models.User{
		UUID:        uuid.New(),
		Name:        "Administrator",
		Username:    username,
		Password:    string(hashedPassword),
		PhoneNumber: phoneNumber,
		Email:       email,
		RoleID:      constants.Admin,
	}

	err = db.FirstOrCreate(&user, models.User{Username: user.Username}).Error
	if err != nil {
		logrus.Errorf("failed to seed admin user: %v", err)
		panic(err)
	}
	logrus.Infof("admin user %s successfully seeded", user.Username)
}
