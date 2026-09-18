package config

import (
	"fmt"
	"os"
	"user-service/common/utils"

	"github.com/sirupsen/logrus"
	_ "github.com/spf13/viper/remote"
)

var Config AppConfig

type AppConfig struct {
	Port                  int      `json:"port"`
	AppName               string   `json:"appName"`
	AppEnv                string   `json:"appEnv"`
	SignatureKey          string   `json:"signatureKey"`
	Database              Database `json:"database"`
	RateLimiterMaxRequest float64  `json:"rateLimiterMaxRequest"`
	RateLimiterTimeSecond float64  `json:"rateLimiterTimeSecond"`
	JwtSecretKey          string   `json:"jwtSecretKey"`
	JwtExpirationTime     int      `json:"jwtExpirationTime"`
	// AllowedOrigins is the CORS allow-list of browser origins permitted to call
	// this API. Leave empty to disable cross-origin browser access entirely.
	AllowedOrigins []string `json:"allowedOrigins"`
}

type Database struct {
	Host                  string `json:"host"`
	Port                  int    `json:"port"`
	Name                  string `json:"name"`
	Username              string `json:"username"`
	Password              string `json:"password"`
	MaxOpenConnection     int    `json:"maxOpenConnection"`
	MaxLifetimeConnection int    `json:"maxLifetimeConnection"`
	MaxIdleConnection     int    `json:"maxIdleConnection"`
	MaxIdleTime           int    `json:"maxIdleTime"`
}

// minSecretLength is the minimum length required for security-sensitive
// config values (JWT signing secret, inter-service signature key) so a
// misconfigured/empty secret fails fast at startup instead of silently
// producing forgeable tokens or signatures.
const minSecretLength = 16

const ProductionEnv = "production"

func Init() {
	err := utils.BindFromJSON(&Config, "config.json", ".")
	if err != nil {
		logrus.Infof("failed to bind config json:%v", err)
		err = utils.BindFromConsul(&Config, os.Getenv("CONSUL_HTTP_URL"), os.Getenv("CONSUL_HTTP_PATH"))
		if err != nil {
			panic(err)
		}
	}

	if err := validate(); err != nil {
		panic(fmt.Errorf("invalid configuration: %w", err))
	}
}

func validate() error {
	if Config.Port <= 0 {
		return fmt.Errorf("port must be a positive number")
	}
	if len(Config.SignatureKey) < minSecretLength {
		return fmt.Errorf("signatureKey must be set and at least %d characters", minSecretLength)
	}
	if len(Config.JwtSecretKey) < minSecretLength {
		return fmt.Errorf("jwtSecretKey must be set and at least %d characters", minSecretLength)
	}
	if Config.JwtExpirationTime <= 0 {
		return fmt.Errorf("jwtExpirationTime must be a positive number")
	}
	if Config.RateLimiterMaxRequest <= 0 {
		return fmt.Errorf("rateLimiterMaxRequest must be a positive number")
	}
	if Config.RateLimiterTimeSecond <= 0 {
		return fmt.Errorf("rateLimiterTimeSecond must be a positive number")
	}
	if Config.Database.Host == "" || Config.Database.Name == "" || Config.Database.Username == "" {
		return fmt.Errorf("database host, name, and username must be set")
	}
	if Config.AppEnv == ProductionEnv && len(Config.AllowedOrigins) == 0 {
		logrus.Warn("allowedOrigins is empty in production; all cross-origin browser requests will be blocked")
	}
	return nil
}
