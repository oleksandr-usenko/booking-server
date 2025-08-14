package routes

import (
	"fmt"
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

	var mediaUrls []string
	for _, fileHeader := range files {
		url, err := cloud.HandleFile(fileHeader)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Upload failed: " + err.Error()})
			return
		}
		mediaUrls = append(mediaUrls, url)
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
		Media:       mediaUrls,
	}

	service, err = service.CreateService()
	if err != nil {
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
