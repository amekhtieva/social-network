package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	pb "post_service/proto"
)

func CreateDatabase(dbInfo string, dbName string) error {
	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
	 	return err
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
   
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	if err != nil {
	 	return err
	}
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
	 	return err
	}

	return nil
}   

func main() {
	port := flag.Int("port", 8090, "grpc server port")
	dbHost := flag.String("db-host", "", "hostname of the database")
	dbPort := flag.Int("db-port", 5433, "port of the database")
	dbName := flag.String("db-name", "", "database name")
	dbUsername := flag.String("db-username", "", "database user")
	dbPassword := flag.String("db-password", "", "database password")

	flag.Parse()

	if port == nil {
		fmt.Fprintln(os.Stderr, "Port is required")
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

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable",
		*dbHost, *dbPort, *dbUsername, *dbPassword)

	err := CreateDatabase(psqlInfo, *dbName)
	if err != nil {
		panic("Failed to create database: " + err.Error())
	}

	psqlInfo = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		*dbHost, *dbPort, *dbUsername, *dbPassword, *dbName)

	db, err := gorm.Open(postgres.Open(psqlInfo), &gorm.Config{})
	if err != nil {
		panic("Failed to connect database: " + err.Error())
	}

	db.Migrator().DropTable(&Post{})
	err = db.AutoMigrate(&Post{})
	if err != nil {
		panic("Failed to migrate database: " + err.Error())
	}

	grpc_server := grpc.NewServer()
	pb.RegisterPostServiceServer(grpc_server, &Server{DB: db})

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		panic(fmt.Sprintf("Failed to listen on port %d: %s", *port, err.Error()))
	}

	err = grpc_server.Serve(listener)
	if err != nil {
		panic("Failed to serve: " + err.Error())
	}
}
