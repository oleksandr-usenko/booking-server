package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const secretKey = "dummy_key"
const ACCESS_TOKEN_LIFETIME = time.Minute * 2
const REFRESH_TOKEN_LIFETIME = time.Hour * 24 * 7

func CreateJWT(email string, userId int64, ttl time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email":  email,
		"userId": userId,
		"exp":    time.Now().Add(ttl).Unix(),
	})

	return token.SignedString([]byte(secretKey))
}

func GenerateTokens(email string, userId int64) (accessToken string, refreshToken string, err error) {
	accessToken, err = CreateJWT(email, userId, ACCESS_TOKEN_LIFETIME)
	if err != nil {
		return
	}
	refreshToken, err = CreateJWT(email, userId, REFRESH_TOKEN_LIFETIME) // 7 days
	return
}

func VerifyToken(token string) (int64, string, error) {
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		_, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return 0, "", errors.New("Could not parse the token: " + err.Error())
	}

	tokenIsValid := parsedToken.Valid
	if !tokenIsValid {
		return 0, "", errors.New("invalid token")
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", errors.New("invalid token claims")
	}

	email := claims["email"].(string)
	userId := int64(claims["userId"].(float64))
	return userId, email, nil
}
