package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func Authenticate(req *http.Request) (string, error) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
        return "", errors.New("No authentication token in header")
    }
	jwtToken := strings.TrimPrefix(authHeader, "Bearer ")

    token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodRSA)
        if !ok {
            return "", fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
        }
        return publicKey, nil
    })
	if err != nil {
        return "", err
    }
	if !token.Valid {
		return "", errors.New("Invalid authentication token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
		return "", errors.New("Invalid authentication token")
    }

	username := claims["username"].(string)
	return username, nil
}
