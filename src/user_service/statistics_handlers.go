package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"

	pb "user_service/proto"
)

type Event struct {
	PostId 	 string `json:"postId"`
	Author	 string `json:"author"`
	Username string `json:"username"`
}

func GetPostAutor(postIdStr string) (string, error) {
	postId, err := strconv.ParseUint(postIdStr, 10, 64)
	if err != nil {
		return "", nil
	}
	grpcReq := &pb.GetPostRequest{
		Id:		  postId,
	}
	resp, err := postServiceClient.GetPost(context.Background(), grpcReq)
	if err != nil {
		return "", err
	}
	return resp.Post.Username, nil
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

	postId := mux.Vars(req)["id"]
	author, err := GetPostAutor(postId)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get post: %s", err.Error()), http.StatusBadRequest)
		return
	}

	event := Event{
		PostId:   postId,
		Author:   author,
		Username: username,
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

	postId := mux.Vars(req)["id"]
	author, err := GetPostAutor(postId)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get post: %s", err.Error()), http.StatusBadRequest)
		return
	}

	event := Event{
		PostId:   postId,
		Author:   author,
		Username: username,
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

func GetPostStatistics(w http.ResponseWriter, req *http.Request) {
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

	params := mux.Vars(req)
	postId, err := strconv.ParseUint(params["id"], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &pb.GetPostStatisticsRequest{
		PostId: postId,
	}

	resp, err := statisticsServiceClient.GetPostStatistics(context.Background(), grpcReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get post statistics: %s", err.Error()), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(resp)
	w.WriteHeader(http.StatusOK)
}

func GetTopPosts(w http.ResponseWriter, req *http.Request) {
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

	grpcReq := &pb.GetTopPostsRequest{}
	orderBy := req.URL.Query().Get("by")
	if orderBy == "likes" {
		grpcReq.OrderBy = pb.GetTopPostsRequest_Likes
	} else if orderBy == "views" {
		grpcReq.OrderBy = pb.GetTopPostsRequest_Views
	} else {
		http.Error(w, "Unknown sort type", http.StatusBadRequest)
	}

	resp, err := statisticsServiceClient.GetTopPosts(context.Background(), grpcReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get top posts: %s", err.Error()), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(resp)
	w.WriteHeader(http.StatusOK)
}

func GetTopUsers(w http.ResponseWriter, req *http.Request) {
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

	resp, err := statisticsServiceClient.GetTopUsers(context.Background(), &empty.Empty{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get top users: %s", err.Error()), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(resp)
	w.WriteHeader(http.StatusOK)
}
