package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"io"

	_ "github.com/lib/pq"
	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserInfo struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	DateOfBirth string `json:"dateOfBirth"`
	Mail       	string `json:"email"`
	Phone       string `json:"phone"`
}

type AuthenticationToken struct {
	Token string `json:"token"`
}

func HashPassword(username string, password string) string {
	hash := md5.Sum([]byte(username + password))
	return hex.EncodeToString(hash[:])
}

func RegisterUser(w http.ResponseWriter, req *http.Request) {
	body := make([]byte, req.ContentLength)
	_, err := req.Body.Read(body)
	defer req.Body.Close()
	if err != io.EOF {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := User{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var exists bool
    db.QueryRow("SELECT exists (SELECT 1 FROM users WHERE username=$1)", user.Username).Scan(&exists)
    if exists {
        http.Error(w, "Username already exists", http.StatusConflict)
        return
    }

	passwordHash := HashPassword(user.Username, user.Password)
	_, err = db.Exec("INSERT INTO users(username, password) VALUES($1, $2)", user.Username, passwordHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func LoginUser(w http.ResponseWriter, req *http.Request) {
	body := make([]byte, req.ContentLength)
	_, err := req.Body.Read(body)
	defer req.Body.Close()
	if err != io.EOF {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := User{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var dbUser User
    err = db.QueryRow("SELECT username, password FROM users WHERE username=$1",
		user.Username).Scan(&dbUser.Username, &dbUser.Password)
    if err != nil {
        http.Error(w, "Incorrect username or password", http.StatusForbidden)
        return
    }

	if dbUser.Password != HashPassword(user.Username, user.Password) {
		http.Error(w, "Incorrect username or password", http.StatusForbidden)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"username": user.Username,
	})

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error signing token: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthenticationToken{Token: tokenString})
}

func UpdateUser(w http.ResponseWriter, req *http.Request) {
	username, err := Authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
    }

	body := make([]byte, req.ContentLength)
	_, err = req.Body.Read(body)
	defer req.Body.Close()
	if err != io.EOF {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userInfo := UserInfo{}
	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var dbUsername string
    err = db.QueryRow("SELECT username FROM users WHERE username=$1", username).Scan(&dbUsername)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

	_, err = db.Exec("UPDATE users SET firstname=$1, lastname=$2, dateofbirth=$3, mail=$4, phone=$5 WHERE username=$6",
		userInfo.FirstName, userInfo.LastName, userInfo.DateOfBirth, userInfo.Mail, userInfo.Phone, username)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update user: %s", err.Error()), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
