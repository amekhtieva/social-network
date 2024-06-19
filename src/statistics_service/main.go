package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"net"
	"os"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"

	pb "statistics_service/proto"
)

var db *sql.DB

func Ping(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func CreateDatabase(dbAddress string, dbName string) error {
	var err error
	db, err = sql.Open("clickhouse", dbAddress)
    if err != nil {
        return err
    }
	for i := 0; i < 5; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	err = db.Ping()
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName))
	if err != nil {
		return err
    }

	_, err = db.Exec("DROP TABLE IF EXISTS likes")
	if err != nil {
	 	return err
	}

	_, err = db.Exec("DROP TABLE IF EXISTS views")
	if err != nil {
	 	return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS likes (
			postId 	 UInt64,
			author 	 String,
			username String
		) ENGINE = ReplacingMergeTree()
		ORDER BY (postId, username)
	`)
	if err != nil {
	 	return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS views (
			postId 	 UInt64,
			author 	 String,
			username String
		) ENGINE = ReplacingMergeTree()
		ORDER BY (postId, username)
	`)
	if err != nil {
	 	return err
	}

    return nil
}

func main() {
	httpPort := flag.Int("http-port", 8200, "http server port")
	grpcPort := flag.Int("grpc-port", 8300, "grpc server port")
	dbAddress := flag.String("db-address", "", "address of the database")
	dbName := flag.String("db-name", "", "database name")
	kafkaURL := flag.String("kafka-url", "", "address of the Kafka")

	flag.Parse()

	if httpPort == nil {
		fmt.Fprintln(os.Stderr, "HTTP port is required")
		os.Exit(1)
	}
	if grpcPort == nil {
		fmt.Fprintln(os.Stderr, "gRPC port is required")
		os.Exit(1)
	}
	if dbAddress == nil || *dbAddress == "" {
		fmt.Fprintln(os.Stderr, "Please provide address of the database")
		os.Exit(1)
	}
	if dbName == nil || *dbName == ""  {
		fmt.Fprintln(os.Stderr, "Please provide a database name")
		os.Exit(1)
	}
	if kafkaURL == nil || *kafkaURL == ""  {
		fmt.Fprintln(os.Stderr, "Please provide Kafka address")
		os.Exit(1)
	}

	err := CreateDatabase(*dbAddress, *dbName)
	if err != nil {
		panic("Failed to create database: " + err.Error())
	}
	defer db.Close()

	go ConsumeEvents("likes", *kafkaURL)
 	go ConsumeEvents("views", *kafkaURL)

	r := mux.NewRouter()
	r.HandleFunc("/ping", Ping).Methods("GET")

	go func() {
		err = http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), r)
		if err != nil {
			panic(err)
		}
	}()

	grpc_server := grpc.NewServer()
	pb.RegisterStatisticsServiceServer(grpc_server, nil)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		panic(fmt.Sprintf("Failed to listen on port %d: %s", *grpcPort, err.Error()))
	}

	go func() {
		err = grpc_server.Serve(listener)
		if err != nil {
			panic("Failed to serve: " + err.Error())
		}
	}()
}
