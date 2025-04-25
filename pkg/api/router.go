package api

import (
	"context"
	"github.com/duyhenryer/go-rest-api/pkg/cache"
	"github.com/duyhenryer/go-rest-api/pkg/database"
	"github.com/duyhenryer/go-rest-api/pkg/middleware"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/swaggo/gin-swagger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	docs "github.com/duyhenryer/go-rest-api/docs"
	swaggerFiles "github.com/swaggo/files"
)

func ContextMiddleware(bookRepository BookRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("appCtx", bookRepository)
		c.Next()
	}
}

func SetupRoutes(r *gin.Engine, logger *zap.Logger, mongoCollection *mongo.Collection, db database.Database, redisClient cache.Cache, ctx *context.Context) {
	bookRepository := NewBookRepository(db, redisClient, ctx)
	userRepository := NewUserRepository(db, ctx)

	r.Use(ContextMiddleware(bookRepository))
	r.Use(middleware.Logger(logger, mongoCollection))
	if gin.Mode() == gin.ReleaseMode {
		r.Use(middleware.Security())
		r.Use(middleware.Xss())
	}
	r.Use(middleware.Cors())

	// Áp dụng RateLimiter chỉ cho nhóm /api/v1
	docs.SwaggerInfo.BasePath = "/api/v1"
	v1 := r.Group("/api/v1")
	v1.Use(middleware.RateLimiter(rate.Every(1*time.Minute), 60)) // Di chuyển vào đây
	{
		v1.GET("/", bookRepository.Healthcheck)
		v1.GET("/books", middleware.APIKeyAuth(), bookRepository.FindBooks)
		v1.POST("/books", middleware.APIKeyAuth(), middleware.JWTAuth(), bookRepository.CreateBook)
		v1.GET("/books/:id", middleware.APIKeyAuth(), bookRepository.FindBook)
		v1.PUT("/books/:id", middleware.APIKeyAuth(), bookRepository.UpdateBook)
		v1.DELETE("/books/:id", middleware.APIKeyAuth(), bookRepository.DeleteBook)
		v1.POST("/login", middleware.APIKeyAuth(), userRepository.LoginHandler)
		v1.POST("/register", middleware.APIKeyAuth(), userRepository.RegisterHandler)
	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func NewRouter(logger *zap.Logger, mongoCollection *mongo.Collection, db database.Database, redisClient cache.Cache, ctx *context.Context) *gin.Engine {
	r := gin.Default()
	SetupRoutes(r, logger, mongoCollection, db, redisClient, ctx)
	return r
}