package routes

import (
	"example.com/middlewares"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(server *gin.Engine) {
	// Health check endpoint
	server.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := server.Group("/api")
	api.GET("/events", getEvents)
	api.GET("/events/:id", getEvent)
	api.GET("/services/:alias", getServicesByAlias)
	api.GET("/schedule/:alias/:date", getScheduleByAliasForDate)
	api.POST("/appointments/:alias", createAppointment)

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
	authenticated.PATCH("/services/:id/add-media", addServiceMedia)
	authenticated.DELETE("/services/:id/delete-media/:mediaId", deleteServiceMedia)
	authenticated.PATCH("/services/:id/update-media-order", updateMediaOrder)
	authenticated.PUT("/services/:id", editService)
	authenticated.DELETE("/services/:id", deleteService)

	// authenticated.GET("/cloudinary-signature", cloud.GetCloudinarySignature)
	// authenticated.POST("/upload", cloud.UploadHandler)

	// User's own schedule endpoints (authenticated)
	authenticated.GET("/schedule/me", getSchedule)
	authenticated.GET("/schedule/me/:date", getScheduleForDate)
	authenticated.POST("/schedule/me/:date", saveSchedule)

	// Appointments (authenticated)
	authenticated.GET("/appointments", getAppointments)
	authenticated.DELETE("/appointments/:id", deleteAppointment)

	// Alias management (authenticated)
	authenticated.GET("/alias", getAlias)
	authenticated.PUT("/alias", updateAlias)
}
