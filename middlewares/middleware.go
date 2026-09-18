package middlewares

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"user-service/common/response"
	"user-service/config"
	"user-service/constants"
	errConstant "user-service/constants/error"
	services "user-service/services/user"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

func HandlePanic() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("Recovered from panic: %v", r)
				c.JSON(http.StatusInternalServerError, response.Response{
					Status:  constants.Error,
					Message: errConstant.ErrInternalServerError.Error(),
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

func RateLimiter(limit *limiter.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := tollbooth.LimitByRequest(limit, c.Writer, c.Request)
		if err != nil {
			c.JSON(http.StatusTooManyRequests, response.Response{
				Status:  constants.Error,
				Message: errConstant.ErrTooManyRequests.Error(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// CORS only reflects the Access-Control-Allow-Origin header back for origins
// present in allowedOrigins; any other origin gets no CORS headers at all, so
// browsers block cross-origin access to the response. Configure allowedOrigins
// explicitly (config.json "allowedOrigins") instead of using a wildcard, since
// this API accepts an Authorization header.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowed[origin]; ok {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-service-name, x-apikey, x-request-at")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// extractBearerToken strictly requires the "Bearer " scheme prefix (RFC 6750),
// rather than merely checking whether the header contains the substring
// "Bearer" anywhere.
func extractBearerToken(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", errConstant.ErrUnauthorized
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", errConstant.ErrUnauthorized
	}
	return token, nil
}

func responseUnauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, response.Response{
		Status:  constants.Error,
		Message: message,
	})
	c.Abort()
}

func validateAPIKey(c *gin.Context) error {
	apiKey := c.GetHeader(constants.XApiKey)
	requestAt := c.GetHeader(constants.XRequestAt)
	serviceName := c.GetHeader(constants.XServiceName)

	signatureKey := config.Config.SignatureKey

	validateKey := fmt.Sprintf("%s:%s:%s", serviceName, signatureKey, requestAt)
	expected := sha256.Sum256([]byte(validateKey))

	// Compare in constant time to avoid leaking information about the
	// expected signature through response-time differences.
	actual, err := hex.DecodeString(apiKey)
	if err != nil || subtle.ConstantTimeCompare(actual, expected[:]) != 1 {
		return errConstant.ErrUnauthorized
	}
	return nil
}

func validateBearerToken(c *gin.Context, header string) error {
	tokenString, err := extractBearerToken(header)
	if err != nil {
		return err
	}

	claims := &services.Claims{}
	tokenJwt, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errConstant.ErrUnauthorized
		}
		jwtSecret := []byte(config.Config.JwtSecretKey)
		return jwtSecret, nil
	})
	if err != nil || !tokenJwt.Valid {
		return errConstant.ErrUnauthorized
	}
	userLogin := c.Request.WithContext(context.WithValue(c.Request.Context(), constants.UserLogin, claims.User))
	c.Request = userLogin
	c.Set(constants.Token, header)
	return nil
}

func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader(constants.Authorization)
		if token == "" {
			responseUnauthorized(c, errConstant.ErrUnauthorized.Error())
			return
		}
		if err := validateBearerToken(c, token); err != nil {
			responseUnauthorized(c, err.Error())
			return
		}
		if err := validateAPIKey(c); err != nil {
			responseUnauthorized(c, err.Error())
			return
		}
		c.Next()
	}
}
