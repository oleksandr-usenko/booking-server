package routes

import (
	"net/http"
	"strconv"
	"time"

	"example.com/models"
	"github.com/gin-gonic/gin"
)

func getSchedule(c *gin.Context) {
	userID := c.GetInt64("userId")

	// Parse `days` from query param, default = 1
	daysStr := c.Query("days")
	days := 1
	if daysStr != "" {
		if v, err := strconv.Atoi(daysStr); err == nil && v > 0 {
			days = v
		}
	}

	start := time.Now().UTC() // still UTC, adjust if you store tz

	out, err := models.GetScheduleForRange(c.Request.Context(), userID, start, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "server error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"start":  start.Format("2006-01-02"),
		"days":   days,
		"ranges": out, // map[date][]TimeRange
	})
}

func getScheduleForDate(c *gin.Context) {
	userID := c.GetInt64("userId")
	dateStr := c.Param("date") // expect /schedule/:date (YYYY-MM-DD)
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid date (use YYYY-MM-DD)"})
		return
	}

	out, err := models.GetSchedule(c.Request.Context(), userID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "server error: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"date":   dateStr,
		"ranges": out,
	})
}

func getScheduleByIdForDate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Could not parse id: " + err.Error()})
		return
	}
	dateStr := c.Param("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid date (use YYYY-MM-DD)"})
		return
	}
	out, err := models.GetSchedule(c.Request.Context(), id, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "server error: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ranges": out})
}

func saveSchedule(c *gin.Context) {
	userID := c.GetInt64("userId")
	dateStr := c.Param("date") // expect /schedule/:date
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid date (use YYYY-MM-DD)"})
		return
	}

	var payload []models.TimeRangePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body: " + err.Error()})
		return
	}

	inserted, err := models.SaveSchedule(c.Request.Context(), userID, date, payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}) // validation errors included
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "schedule saved",
		"date":    dateStr,
		"ranges":  inserted,
	})
}
