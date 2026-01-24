package routes

import (
	"net/http"
	"strings"

	"example.com/models"
	"github.com/gin-gonic/gin"
)

func createAppointment(c *gin.Context) {
	alias := c.Param("alias")

	user, err := models.GetUserByAlias(alias)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
		return
	}

	var appt models.Appointment
	if err := c.ShouldBindJSON(&appt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body: " + err.Error()})
		return
	}

	// Validate service belongs to this user
	service, err := models.GetServiceById(appt.ServiceID, user.ID)
	if err != nil || service == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "service not found for this user"})
		return
	}

	appt.UserID = user.ID

	err = models.CreateAppointment(c.Request.Context(), &appt)
	if err != nil {
		if strings.Contains(err.Error(), "no available timeslot") {
			c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create appointment: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "appointment created",
		"appointment": appt,
	})
}

func getAppointments(c *gin.Context) {
	userID := c.GetInt64("userId")

	appointments, err := models.GetAppointments(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get appointments: " + err.Error()})
		return
	}

	if appointments == nil {
		appointments = []models.Appointment{}
	}

	c.JSON(http.StatusOK, gin.H{"appointments": appointments})
}

func deleteAppointment(c *gin.Context) {
	userID := c.GetInt64("userId")
	appointmentID := c.Param("id")

	err := models.DeleteAppointment(c.Request.Context(), appointmentID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete appointment: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "appointment deleted"})
}
