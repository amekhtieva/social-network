package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

type Event struct {
	PostId 	 string `json:"postId"`
	Username string `json:"username"`
}

func Like(w http.ResponseWriter, req *http.Request) {
	username, err := Authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
    }

	err = CheckUserExists(username)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

	event := Event{
		Username: username,
		PostId:   req.URL.Query().Get("post"),
	}

	msg, err := json.Marshal(event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
 	}

	err = kafkaLikeWriter.WriteMessages(context.Background(), kafka.Message{
		Value: msg,
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to send message to Kafka: %s", err.Error()), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func View(w http.ResponseWriter, req *http.Request) {
	username, err := Authenticate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
    }

	err = CheckUserExists(username)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

	event := Event{
		Username: username,
		PostId:   req.URL.Query().Get("post"),
	}

	msg, err := json.Marshal(event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
 	}

	err = kafkaViewWriter.WriteMessages(context.Background(), kafka.Message{
		Value: msg,
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to send message to Kafka: %s", err.Error()), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
