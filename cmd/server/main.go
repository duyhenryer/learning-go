package main

import (
	"context"
	"github.com/duyhenryer/go-rest-api/pkg/api"
	"github.com/duyhenryer/go-rest-api/pkg/cache"
	"github.com/duyhenryer/go-rest-api/pkg/database"
	"github.com/duyhenryer/go-rest-api/pkg/middleware"
	"log"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)
// @title           Swagger Example API
// @version         1.0
// @description     This is a sample server celler server.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8001
// @BasePath  /api/v1

// @securityDefinitions.apikey JwtAuth
// @in header
// @name Authorization

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	redisClient := cache.NewRedisClient()
	db := database.NewDatabase()
	dbWrapper := &database.GormDatabase{DB: db}
	mongoCollection := database.SetupMongoDB()
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	gin.SetMode(gin.DebugMode)

	// Khởi tạo router
	r := gin.New()
	r.Use(middleware.PrometheusMiddleware()) // Áp dụng Prometheus trước

	// Thêm route vào router hiện tại
	api.SetupRoutes(r, logger, mongoCollection, dbWrapper, redisClient, &ctx)

	// Đăng ký endpoint /metrics
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	if err := r.Run(":8001"); err != nil {
		log.Fatal(err)
	}
}