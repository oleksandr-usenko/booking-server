package routes

import (
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
	authenticated.PUT("/services/:id", editService)
	authenticated.PATCH("/services/:id/add-media", addServiceMedia)
	authenticated.DELETE("/services/:serviceId/delete-media/:id", deleteServiceMedia)
	authenticated.PATCH("/services/:id/update-media-order", updateMediaOrder)

	// authenticated.GET("/cloudinary-signature", cloud.GetCloudinarySignature)
	// authenticated.POST("/upload", cloud.UploadHandler)

	authenticated.GET("/schedule", getSchedule)
	authenticated.GET("/schedule/:date", getScheduleForDate)
	authenticated.POST("/schedule/:date", saveSchedule)
}
