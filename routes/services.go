package routes

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"example.com/cloud"
	"example.com/models"

	"github.com/gin-gonic/gin"
)

func getServicesForUser(context *gin.Context) {
	userId := context.GetInt64("userId")
	services, err := models.GetServicesForUser(userId)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Server error: " + err.Error()})
		return
	}
	context.JSON(http.StatusOK, services)
}

func getServicesByAlias(c *gin.Context) {
	alias := c.Param("alias")

	user, err := models.GetUserByAlias(alias)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
		return
	}

	services, err := models.GetServicesForUser(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Could not fetch services: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, services)
}

func createService(context *gin.Context) {
	err := context.Request.ParseMultipartForm(10 << 20) // 10 MB max memory
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid multipart form"})
		return
	}

	name := context.PostForm("name")
	description := context.PostForm("description")
	currency := context.PostForm("currency")
	priceStr := context.PostForm("price")
	durationStr := context.PostForm("duration")
	price, _ := strconv.ParseInt(priceStr, 10, 64)
	duration, _ := strconv.ParseInt(durationStr, 10, 64)

	userId := context.GetInt64("userId")

	form, _ := context.MultipartForm()
	files := form.File["media"]

	var mediaItems []models.MediaItem
	for _, fileHeader := range files {
		item, err := cloud.HandleFile(fileHeader)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Upload failed: " + err.Error()})
			return
		}
		mediaItems = append(mediaItems, item)
	}

	now := time.Now().UTC()
	service := &models.Service{
		Name:        name,
		Description: description,
		Price:       price,
		Currency:    currency,
		Duration:    duration,
		Timestamp:   &now,
		UserID:      userId,
		Media:       mediaItems,
	}

	service, err = service.CreateService()
	if err != nil {
		log.Printf("CreateService error - Failed to create service: %v", err)
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Server error: " + err.Error()})
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": service})
}

func editService(context *gin.Context) {
	id, err := strconv.ParseInt(context.Param("id"), 10, 64)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Could not parse id: " + err.Error()})
		return
	}

	userId := context.GetInt64("userId")
	service, err := models.GetServiceById(id, userId)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Could not fetch service by id: " + err.Error()})
		return
	}

	if userId != service.UserID {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Could not authorize user"})
		return
	}

	var updatedService models.Service
	err = context.ShouldBindJSON(&updatedService)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "can't parse the request: " + err.Error()})
		return
	}

	updatedService.ID = id
	updatedService.UserID = userId
	updatedService.Media = service.Media
	err = updatedService.UpdateService()
	if err != nil {
		fmt.Println(err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Could not update event by id: " + err.Error()})
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Success!", "service": updatedService})
}

func deleteServiceMedia(c *gin.Context) {
	serviceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid service ID: " + err.Error()})
		return
	}

	publicID := c.Param("mediaId")
	if publicID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Missing media id"})
		return
	}

	userID := c.GetInt64("userId")

	service, err := models.GetServiceById(serviceID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Service not found"})
		return
	}

	if service.UserID != userID {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "You are not allowed to modify this service"})
		return
	}

	cloud.DeleteMedia(c, publicID)

	updatedMedia := []models.MediaItem{}
	for _, mediaItem := range service.Media {
		if mediaItem.PublicID != publicID {
			updatedMedia = append(updatedMedia, mediaItem)
		}
	}
	service.Media = updatedMedia

	_, err = service.SaveMedia()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update service media: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Media deleted successfully"})

}

type UpdateServiceMedia struct {
	Media []models.MediaItem `json:"media"`
}

func addServiceMedia(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("userId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid service ID: " + err.Error()})
		return
	}

	service, err := models.GetServiceById(id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Service not found " + err.Error()})
		return
	}

	if service.UserID != userID {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "You are not allowed to modify this service"})
		return
	}

	form, _ := c.MultipartForm()
	files := form.File["media"]

	var mediaItems []models.MediaItem
	for _, fileHeader := range files {
		item, err := cloud.HandleFile(fileHeader)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload failed: " + err.Error()})
			return
		}
		mediaItems = append(mediaItems, item)
	}

	service.Media = append(service.Media, mediaItems...)

	service, err = service.SaveMedia()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update service media: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Media added successfully", "service": service})
}

func updateMediaOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("userId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid service ID: " + err.Error()})
		return
	}

	service, err := models.GetServiceById(id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Service not found " + err.Error()})
		return
	}

	if service.UserID != userID {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "You are not allowed to modify this service"})
		return
	}

	var updatedMedia UpdateServiceMedia
	err = c.BindJSON(&updatedMedia)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body: " + err.Error()})
		return
	}

	service.Media = updatedMedia.Media
	service, err = service.SaveMedia()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update service media: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Media added successfully", "service": service})
}

func deleteService(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid service ID: " + err.Error()})
		return
	}

	userID := c.GetInt64("userId")

	service, err := models.GetServiceById(id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Service not found: " + err.Error()})
		return
	}

	if service.UserID != userID {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "You are not authorized to delete this service"})
		return
	}

	// Delete all media from Cloudinary before deleting the service
	for _, mediaItem := range service.Media {
		cloud.DeleteMedia(c, mediaItem.PublicID)
	}

	err = service.DeleteService()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete service: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Service deleted successfully"})
}
