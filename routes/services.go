package routes

import (
	"net/http"

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
	service := new(models.Service)
	err := context.ShouldBindJSON(&service)

	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "can't parse the request" + err.Error()})
		return
	}

	userId := context.GetInt64("userId")
	service.UserID = userId

	service, err = service.CreateService()
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Server error: " + err.Error()})
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": service})
}
