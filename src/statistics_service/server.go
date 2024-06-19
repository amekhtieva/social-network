package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/segmentio/kafka-go"

	pb "statistics_service/proto"
)

type Event struct {
	PostId 	 string `json:"postId"`
	Author	 string `json:"author"`
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

		_, err = db.Exec(fmt.Sprintf(
			"INSERT INTO %s (postId, author, username) VALUES (%s, '%s', '%s')",
			topic,
			event.PostId,
			event.Author,
			event.Username,
		))
		if err != nil {
			log.Printf("Failed to write %s: %s", topic, err)
			continue
		}
	}
}

func (s *Server) GetPostStatistics(
	ctx context.Context,
	req *pb.GetPostStatisticsRequest,
) (*pb.GetPostStatisticsResponse, error) {
	postStatistics := pb.GetPostStatisticsResponse{
		PostId: req.PostId,
	}
	
	err := db.QueryRow("SELECT COUNT(*) FROM likes FINAL WHERE postId=$1", req.PostId).Scan(&postStatistics.Likes)
	if err == sql.ErrNoRows {
		postStatistics.Likes = 0
	} else if err != nil {
		return nil, err
	}

	err = db.QueryRow("SELECT COUNT(*) FROM views FINAL WHERE postId=$1", req.PostId).Scan(&postStatistics.Views)
	if err == sql.ErrNoRows {
		postStatistics.Views = 0
	} else if err != nil {
		return nil, err
	}

	return &postStatistics, nil
}

func (s *Server) GetTopPosts(
	ctx context.Context,
	req *pb.GetTopPostsRequest,
) (*pb.GetTopPostsResponse, error) {
	var table string
	if req.OrderBy == pb.GetTopPostsRequest_Likes {
		table = "likes"
	} else if req.OrderBy == pb.GetTopPostsRequest_Views {
		table = "views"
	} else {
		return nil, errors.New("Unknown sort type")
	}

	query := fmt.Sprintf(`
		SELECT postId, author, COUNT(*) as statistics
		FROM %s FINAL
		GROUP BY postId, author
		ORDER BY statistics DESC
		LIMIT 5
	`, table)

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*pb.GetTopPostsResponse_Post
	for rows.Next() {
		var post pb.GetTopPostsResponse_Post
		err = rows.Scan(&post.Id, &post.Author, &post.Statistics)
		if err != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return &pb.GetTopPostsResponse{Posts: posts}, nil
}

func (s *Server) GetTopUsers(
	ctx context.Context,
	req *empty.Empty,
) (*pb.GetTopUsersResponse, error) {
	query := `
		SELECT author, COUNT(*) as statistics
		FROM likes FINAL
		GROUP BY author
		ORDER BY statistics DESC
  		LIMIT 3
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*pb.GetTopUsersResponse_User
	for rows.Next() {
		var user pb.GetTopUsersResponse_User
		err = rows.Scan(&user.Username, &user.Likes)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return &pb.GetTopUsersResponse{Users: users}, nil
}
