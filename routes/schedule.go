package routes

import (
	"net/http"
	"time"

	"example.com/models"
	"github.com/gin-gonic/gin"
)

func getSchedule(c *gin.Context) {
	userID := c.GetInt64("userId")

	// FIX: exported field + json tag so Gin can bind it
	type DaysRange struct {
		Days int `json:"days" binding:"required"` // e.g. 1,4,7,14,30,60
	}
	var req DaysRange
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Can't parse the request: " + err.Error()})
		return
	}
	if req.Days <= 0 {
		req.Days = 1 // default to 1 day (today only)
	}

	start := time.Now().UTC() // or use the user's tz if you store local dates

	out, err := models.GetScheduleForRange(c.Request.Context(), userID, start, req.Days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "server error: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"start":  start.Format("2006-01-02"),
		"days":   req.Days,
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
