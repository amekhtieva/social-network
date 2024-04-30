package main

import (
	"crypto/md5"
	"crypto/rsa"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
 	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserInfo struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	DateOfBirth string `json:"dateOfBirth"`
	Mail       string `json:"email"`
	Phone       string `json:"phone"`
}

type AuthenticationToken struct {
	Token string `json:"token"`
}

var db *sql.DB
var publicKey  *rsa.PublicKey
var privateKey *rsa.PrivateKey

func hashPassword(username string, password string) string {
	hash := md5.Sum([]byte(username + password))
	return hex.EncodeToString(hash[:])
}

func registerUser(w http.ResponseWriter, req *http.Request) {
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

	passwordHash := hashPassword(user.Username, user.Password)
	_, err = db.Exec("INSERT INTO users(username, password) VALUES($1, $2)", user.Username, passwordHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func loginUser(w http.ResponseWriter, req *http.Request) {
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

	if dbUser.Password != hashPassword(user.Username, user.Password) {
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

func updateUser(w http.ResponseWriter, req *http.Request) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
        http.Error(w, "No authentication token in header", http.StatusUnauthorized)
        return
    }
	jwtToken := strings.TrimPrefix(authHeader, "Bearer ")

    token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodRSA)
        if !ok {
            return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
        }
        return publicKey, nil
    })
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
        return
    }
	if !token.Valid {
		http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
		http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
		return
    }

	username := claims["username"].(string)

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

func main() {
	privateFile := flag.String("private", "", "path to JWT private key `file`")
	publicFile := flag.String("public", "", "path to JWT public key `file`")
	port := flag.Int("port", 8080, "http server port")
	dbHost := flag.String("db-host", "", "hostname of the database")
	dbPort := flag.Int("db-port", 5432, "port of the database")
	dbName := flag.String("db-name", "", "database name")
	dbUsername := flag.String("db-username", "", "database user")
	dbPassword := flag.String("db-password", "", "database password")

	flag.Parse()

	if port == nil {
		fmt.Fprintln(os.Stderr, "Port is required")
		os.Exit(1)
	}
	if privateFile == nil || *privateFile == "" {
		fmt.Fprintln(os.Stderr, "Please provide a path to JWT private key file")
		os.Exit(1)
	}
	if publicFile == nil || *publicFile == "" {
		fmt.Fprintln(os.Stderr, "Please provide a path to JWT public key file")
		os.Exit(1)
	}
	if dbHost == nil || *dbHost == "" {
		fmt.Fprintln(os.Stderr, "Please provide a hostname of the database")
		os.Exit(1)
	}
	if dbPort == nil {
		fmt.Fprintln(os.Stderr, "Please provide a port of the database")
		os.Exit(1)
	}
	if dbName == nil || *dbName == ""  {
		fmt.Fprintln(os.Stderr, "Please provide a database name")
		os.Exit(1)
	}
	if dbUsername == nil || *dbUsername == ""  {
		fmt.Fprintln(os.Stderr, "Please provide a database username")
		os.Exit(1)
	}
	if dbPassword == nil || *dbPassword == ""  {
		fmt.Fprintln(os.Stderr, "Please provide a database password")
		os.Exit(1)
	}

	absolutePrivateFile, err := filepath.Abs(*privateFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	absolutePublicFile, err := filepath.Abs(*publicFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	private, err := os.ReadFile(absolutePrivateFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	public, err := os.ReadFile(absolutePublicFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(private)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	publicKey, err = jwt.ParseRSAPublicKeyFromPEM(public)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable",
		*dbHost, *dbPort, *dbUsername, *dbPassword)
    db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
        panic(err)
    } 
    defer db.Close()

	for i := 0; i < 5; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", *dbName))
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", *dbName))
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("DROP TABLE IF EXISTS users")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`
		CREATE TABLE users (
			id 			SERIAL PRIMARY KEY,
			username 	TEXT NOT NULL,
			password 	TEXT NOT NULL,
			firstName   TEXT,
			lastName    TEXT,
			dateOfBirth TEXT,
			mail       	TEXT,
			phone       TEXT
		)
	`)
	if err != nil {
		panic(err)
	}
	
	r := mux.NewRouter()
	r.HandleFunc("/register", registerUser).Methods("POST")
	r.HandleFunc("/login", loginUser).Methods("POST")
	r.HandleFunc("/update", updateUser).Methods("PUT")

	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), r)
	if err != nil {
		panic(err)
	}
}
