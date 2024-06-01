package main

import (
	"crypto/rsa"
	"database/sql"
	"flag"
	"fmt"
 	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "user_service/proto"
)

var db *sql.DB

var postServiceClient pb.PostServiceClient

var publicKey  *rsa.PublicKey
var privateKey *rsa.PrivateKey

var kafkaLikeWriter *kafka.Writer
var kafkaViewWriter *kafka.Writer

func ConnectToPostService(addr string) error {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	postServiceClient = pb.NewPostServiceClient(conn)
	return nil
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
	postServerAddr := flag.String("post-server-addr", "", "address of the gRPC post server")
	kafkaURL := flag.String("kafka-url", "", "address of the Kafka")

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

	if postServerAddr == nil || *postServerAddr == ""  {
		fmt.Fprintln(os.Stderr, "Please provide a database post server address")
		os.Exit(1)
	}

	if kafkaURL == nil || *kafkaURL == ""  {
		fmt.Fprintln(os.Stderr, "Please provide Kafka address")
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

	err = ConnectToPostService(*postServerAddr)
	if err != nil {
		panic(err)
	}

	kafkaLikeWriter = &kafka.Writer{
		Addr:     kafka.TCP(*kafkaURL),
		Topic:    "likes",
	}
	defer kafkaLikeWriter.Close()

	kafkaViewWriter = &kafka.Writer{
		Addr:     kafka.TCP(*kafkaURL),
		Topic:    "views",
	}
	defer kafkaViewWriter.Close()

	r := mux.NewRouter()

	r.HandleFunc("/user/register", RegisterUser).Methods("POST")
	r.HandleFunc("/user/login", LoginUser).Methods("POST")
	r.HandleFunc("/user/update", UpdateUser).Methods("PUT")

	r.HandleFunc("/post", CreatePost).Methods("POST")
	r.HandleFunc("/post/{id}", UpdatePost).Methods("PUT")
	r.HandleFunc("/post/{id}", DeletePost).Methods("DELETE")
	r.HandleFunc("/post/{id}", GetPost).Methods("GET")
	r.HandleFunc("/posts", ListPosts).Methods("GET")

	r.HandleFunc("/post/{id}/like", Like).Methods("POST")
	r.HandleFunc("/post/{id}/view", View).Methods("POST")

	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), r)
	if err != nil {
		panic(err)
	}
}
