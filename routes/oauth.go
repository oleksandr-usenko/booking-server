package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"example.com/config"
	"example.com/models"
	"example.com/utils"
	"github.com/gin-gonic/gin"
)

// Google OAuth user info response
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// Facebook OAuth user info response
type FacebookUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// googleLogin initiates the Google OAuth flow
func googleLogin(c *gin.Context) {
	url := config.GoogleOAuthConfig.AuthCodeURL(config.OAuthStateString)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// googleCallback handles the callback from Google OAuth
func googleCallback(c *gin.Context) {
	state := c.Query("state")
	if state != config.OAuthStateString {
		redirectWithError(c, "Invalid OAuth state")
		return
	}

	code := c.Query("code")
	if code == "" {
		redirectWithError(c, "Code not found")
		return
	}

	token, err := config.GoogleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		redirectWithError(c, "Failed to exchange token")
		return
	}

	// Get user info from Google
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		redirectWithError(c, "Failed to get user info")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		redirectWithError(c, "Failed to read response")
		return
	}

	var googleUser GoogleUserInfo
	if err := json.Unmarshal(body, &googleUser); err != nil {
		redirectWithError(c, "Failed to parse user info")
		return
	}

	if googleUser.Email == "" {
		redirectWithError(c, "Email not provided by Google")
		return
	}

	// Find or create user
	oauthUser := &models.OAuthUser{
		Email:           googleUser.Email,
		Name:            googleUser.Name,
		OAuthProvider:   "google",
		OAuthProviderID: googleUser.ID,
	}

	user, err := models.FindOrCreateOAuthUser(oauthUser)
	if err != nil {
		redirectWithError(c, "Failed to create user")
		return
	}

	// Generate JWT tokens
	accessToken, refreshToken, err := utils.GenerateTokens(user.Email, user.ID)
	if err != nil {
		redirectWithError(c, "Failed to generate tokens")
		return
	}

	// Redirect to frontend with tokens
	redirectWithTokens(c, accessToken, refreshToken)
}

// facebookLogin initiates the Facebook OAuth flow
func facebookLogin(c *gin.Context) {
	url := config.FacebookOAuthConfig.AuthCodeURL(config.OAuthStateString)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// facebookCallback handles the callback from Facebook OAuth
func facebookCallback(c *gin.Context) {
	state := c.Query("state")
	if state != config.OAuthStateString {
		redirectWithError(c, "Invalid OAuth state")
		return
	}

	code := c.Query("code")
	if code == "" {
		redirectWithError(c, "Code not found")
		return
	}

	token, err := config.FacebookOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		redirectWithError(c, "Failed to exchange token")
		return
	}

	// Get user info from Facebook
	resp, err := http.Get(fmt.Sprintf(
		"https://graph.facebook.com/me?fields=id,email,name&access_token=%s",
		token.AccessToken,
	))
	if err != nil {
		redirectWithError(c, "Failed to get user info")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		redirectWithError(c, "Failed to read response")
		return
	}

	var fbUser FacebookUserInfo
	if err := json.Unmarshal(body, &fbUser); err != nil {
		redirectWithError(c, "Failed to parse user info")
		return
	}

	if fbUser.Email == "" {
		redirectWithError(c, "Email not provided by Facebook. Please ensure email permission is granted.")
		return
	}

	// Find or create user
	oauthUser := &models.OAuthUser{
		Email:           fbUser.Email,
		Name:            fbUser.Name,
		OAuthProvider:   "facebook",
		OAuthProviderID: fbUser.ID,
	}

	user, err := models.FindOrCreateOAuthUser(oauthUser)
	if err != nil {
		redirectWithError(c, "Failed to create user")
		return
	}

	// Generate JWT tokens
	accessToken, refreshToken, err := utils.GenerateTokens(user.Email, user.ID)
	if err != nil {
		redirectWithError(c, "Failed to generate tokens")
		return
	}

	// Redirect to frontend with tokens
	redirectWithTokens(c, accessToken, refreshToken)
}

// Mobile token endpoints

type GoogleTokenRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

type FacebookTokenRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

// googleTokenLogin verifies a Google ID token from a mobile app and returns JWTs
func googleTokenLogin(c *gin.Context) {
	var req GoogleTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "id_token is required"})
		return
	}

	// Verify the ID token with Google
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + req.IDToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to verify token"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid ID token"})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to read token info"})
		return
	}

	var tokenInfo struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
		Aud           string `json:"aud"`
	}
	if err := json.Unmarshal(body, &tokenInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse token info"})
		return
	}

	// Verify the token was issued for our app
	if tokenInfo.Aud != config.GoogleOAuthConfig.ClientID {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Token was not issued for this application"})
		return
	}

	if tokenInfo.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email not available in token"})
		return
	}

	oauthUser := &models.OAuthUser{
		Email:           tokenInfo.Email,
		Name:            tokenInfo.Name,
		OAuthProvider:   "google",
		OAuthProviderID: tokenInfo.Sub,
	}

	user, err := models.FindOrCreateOAuthUser(oauthUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create user"})
		return
	}

	accessToken, refreshToken, err := utils.GenerateTokens(user.Email, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// facebookTokenLogin verifies a Facebook access token from a mobile app and returns JWTs
func facebookTokenLogin(c *gin.Context) {
	var req FacebookTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "access_token is required"})
		return
	}

	// Verify and get user info from Facebook
	resp, err := http.Get(fmt.Sprintf(
		"https://graph.facebook.com/me?fields=id,email,name&access_token=%s",
		req.AccessToken,
	))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to verify token"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid access token"})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to read response"})
		return
	}

	var fbUser FacebookUserInfo
	if err := json.Unmarshal(body, &fbUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse user info"})
		return
	}

	if fbUser.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email not provided by Facebook"})
		return
	}

	oauthUser := &models.OAuthUser{
		Email:           fbUser.Email,
		Name:            fbUser.Name,
		OAuthProvider:   "facebook",
		OAuthProviderID: fbUser.ID,
	}

	user, err := models.FindOrCreateOAuthUser(oauthUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create user"})
		return
	}

	accessToken, refreshToken, err := utils.GenerateTokens(user.Email, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Web redirect helpers

// redirectWithError redirects to frontend with error message
func redirectWithError(c *gin.Context, errorMsg string) {
	redirectURL := fmt.Sprintf("%s/auth/callback?error=%s",
		config.FrontendURL,
		url.QueryEscape(errorMsg),
	)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// redirectWithTokens redirects to frontend with tokens
func redirectWithTokens(c *gin.Context, accessToken, refreshToken string) {
	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s&refresh_token=%s",
		config.FrontendURL,
		url.QueryEscape(accessToken),
		url.QueryEscape(refreshToken),
	)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}
