package routes

import (
	"net/http"
	"strconv"

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

func createService(context *gin.Context) {
	err := context.Request.ParseMultipartForm(10 << 20) // 10 MB max memory
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid multipart form"})
		return
	}

	name := context.PostForm("name")
	description := context.PostForm("description")
	priceStr := context.PostForm("price")
	durationStr := context.PostForm("duration")
	price, _ := strconv.ParseInt(priceStr, 10, 64)
	duration, _ := strconv.ParseInt(durationStr, 10, 64)

	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "can't parse the request" + err.Error()})
		return
	}

	userId := context.GetInt64("userId")

	form, _ := context.MultipartForm()
	files := form.File["media"]

	var mediaUrls []string
	for _, fileHeader := range files {
		url, err := cloud.HandleFile(fileHeader)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Upload failed: " + err.Error()})
			return
		}
		mediaUrls = append(mediaUrls, url)
	}

	service := &models.Service{
		Name:        name,
		Description: description,
		Price:       price,
		Duration:    duration,
		UserID:      userId,
		Media:       mediaUrls, // assuming Media is []string
	}

	service, err = service.CreateService()
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Server error: " + err.Error()})
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": service})
}
