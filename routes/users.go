package routes

import (
	"log"
	"net/http"
	"strings"

	"example.com/models"
	"example.com/utils"
	"github.com/gin-gonic/gin"
)

func getAlias(c *gin.Context) {
	userID := c.GetInt64("userId")

	alias, err := models.GetAlias(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get alias: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alias": alias})
}

func updateAlias(c *gin.Context) {
	userID := c.GetInt64("userId")

	var body struct {
		Alias string `json:"alias" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body: " + err.Error()})
		return
	}

	err := models.UpdateAlias(userID, body.Alias)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"message": "alias already taken"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update alias: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "alias updated", "alias": body.Alias})
}

func signup(context *gin.Context) {
	var user models.User
	err := context.ShouldBindJSON(&user)

	if err != nil {
		log.Printf("Signup error - Failed to parse request: %v", err)
		context.JSON(http.StatusBadRequest, gin.H{"message": "can't parse the request: " + err.Error()})
		return
	}

	err = user.Save()
	if err != nil {
		log.Printf("Signup error - Failed to save user: %v", err)
		// Check if it's a duplicate email error
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			context.JSON(http.StatusConflict, gin.H{"message": "Email already exists"})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Server error: " + err.Error()})
		return
	}

	log.Printf("User created successfully: %s", user.Email)
	context.JSON(http.StatusOK, gin.H{"message": "User created"})
}

func login(context *gin.Context) {
	var user models.User
	err := context.ShouldBindJSON(&user)

	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "can't parse the request: " + err.Error()})
		return
	}

	err = user.ValidateCredentials()

	if err != nil {
		context.JSON(http.StatusUnauthorized, gin.H{
			"message":    "Could not authorize user: invalid login and/or password",
			"error_type": "invalidLogin",
		})
		return
	}

	token, refreshToken, err := utils.GenerateTokens(user.Email, user.ID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Could not authenticate the user: " + err.Error()})
		return
	}

	// http.SetCookie(context.Writer, &http.Cookie{
	// 	Name:     "refresh_token",
	// 	Value:    refreshToken,
	// 	Path:     "/",
	// 	MaxAge:   int(utils.REFRESH_TOKEN_LIFETIME.Seconds()),
	// 	HttpOnly: true,
	// 	Secure:   false,                // because localhost HTTP
	// 	SameSite: http.SameSiteLaxMode, // or http.SameSiteDefaultMode, but not None for HTTP
	// })

	context.SetCookie("refresh_token", refreshToken, int(utils.REFRESH_TOKEN_LIFETIME), "/", "localhost", false, true)
	context.JSON(http.StatusOK, gin.H{
		"message":              "Auth success",
		"token":                token,
		"refresh_token":        refreshToken,
		"refresh_token_expire": int(utils.REFRESH_TOKEN_LIFETIME),
	})
}

func refresh(context *gin.Context) {
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := context.BindJSON(&body); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}
	refreshToken := body.RefreshToken

	userId, email, err := utils.VerifyToken(refreshToken)
	if err != nil {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid refresh token"})
		return
	}

	accessToken, refreshToken, err := utils.GenerateTokens(email, userId)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Token generation error"})
		return
	}

	context.SetCookie("refresh_token", refreshToken, int(utils.REFRESH_TOKEN_LIFETIME), "/", "localhost", false, true)
	context.JSON(http.StatusOK, gin.H{
		"accessToken": accessToken,
	})
}
