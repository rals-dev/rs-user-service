package cmd

import (
	"fmt"
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"time"
	"user-service/common/response"
	"user-service/config"
	"user-service/constants"
	"user-service/controllers"
	"user-service/database/seeders"
	"user-service/domain/models"
	"user-service/middlewares"
	"user-service/repositories"
	"user-service/routes"
	"user-service/services"
)

var command = &cobra.Command{
	Use:   "serve",
	Short: "Start the server",
	Run: func(cmd *cobra.Command, args []string) {
		_ = godotenv.Load()
		config.Init()
		db, err := config.InitDatabase()
		if err != nil {
			panic(err)
		}
		timezone := os.Getenv("TIMEZONE")
		if timezone == "" {
			timezone = "Asia/Jakarta"
		}
		loc, err := time.LoadLocation(timezone)

		if err != nil {
			panic(err)
		}
		time.Local = loc

		err = db.AutoMigrate(
			&models.Role{}, &models.User{},
		)
		if err != nil {
			panic(err)
		}

		seeders.NewSeederRegistry(db).Run()

		repository := repositories.NewRepositoryRegistry(db)
		service := services.NewServiceRegistry(repository)

		controller := controllers.NewControllerRegistry(service)

		// gin.New() (instead of gin.Default()) avoids registering gin's built-in
		// Recovery() middleware on top of middlewares.HandlePanic(), which
		// already recovers panics and would otherwise never be reached.
		router := gin.New()
		router.Use(gin.Logger())
		router.Use(middlewares.HandlePanic())

		router.Use(middlewares.CORS(config.Config.AllowedOrigins))

		lmt := tollbooth.NewLimiter(
			config.Config.RateLimiterMaxRequest,
			&limiter.ExpirableOptions{
				DefaultExpirationTTL: time.Duration(config.Config.RateLimiterTimeSecond) * time.Second,
			},
		)
		router.Use(middlewares.RateLimiter(lmt))

		router.NoRoute(func(c *gin.Context) {
			c.JSON(http.StatusNotFound, response.Response{
				Status:  constants.Error,
				Message: fmt.Sprintf("Path %s", http.StatusText(http.StatusNotFound)),
			})
		})
		router.GET("/", func(context *gin.Context) {
			context.JSON(http.StatusOK, response.Response{
				Status:  constants.Success,
				Message: "Welcome to User Service",
			})
		})

		group := router.Group("/api/v1")
		route := routes.NewRouteRegistry(controller, group)
		route.Serve()

		port := fmt.Sprintf(":%d", config.Config.Port)
		err = router.Run(port)
		if err != nil {
			panic(err)
		}
	},
}

func Run() {
	err := command.Execute()
	if err != nil {
		panic(err)
	}
}
