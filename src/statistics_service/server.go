package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

type Event struct {
	PostId 	 string `json:"postId"`
	Username string `json:"username"`
}

func ConsumeEvents(topic string, kafkaURL string) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaURL},
		Topic:    topic,
	})
	defer reader.Close()
   
	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Failed to read message from Kafka %s: %s", topic, err)
			continue
		}
	
		var event Event
		err = json.Unmarshal(msg.Value, &event)
		if err != nil {
			log.Printf("Failed to deserialize message: %s", err)
			continue
		}

		_, err = db.Exec(fmt.Sprintf("INSERT INTO %s (postId, username) VALUES (%s, '%s')", topic, event.PostId, event.Username))
		if err != nil {
			log.Printf("Failed to write %s: %s", topic, err)
			continue
		}
	}
}
