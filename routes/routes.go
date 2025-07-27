package routes

import (
	"example.com/cloud"
	"example.com/middlewares"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(server *gin.Engine) {
	api := server.Group("/api")
	api.GET("/events", getEvents)
	api.GET("/events/:id", getEvent)

	auth := api.Group("/auth")
	auth.POST("/signup", signup)
	auth.POST("/login", login)
	auth.POST("/refresh", refresh)

	authenticated := api.Group("/")
	authenticated.Use(middlewares.Authenticate)
	authenticated.POST("/events", createEvent)
	authenticated.PUT("/events/:id", updateEvent)
	authenticated.DELETE("/events/:id", deleteEvent)
	authenticated.POST("/events/:id/register", registerEvent)
	authenticated.DELETE("/events/:id/register", unregisterEvent)

	authenticated.GET("/services", getServicesForUser)
	authenticated.POST("/services", createService)

	// authenticated.GET("/cloudinary-signature", cloud.GetCloudinarySignature)
	authenticated.POST("/upload", cloud.UploadHandler)
}
