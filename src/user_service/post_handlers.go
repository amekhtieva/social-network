package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
	"github.com/gorilla/mux"

	pb "user_service/proto"
)

type PostContent struct {
	Content string `json:"content"`
}

func CheckUserExists(username string) error {
	var userId uint64
    err := db.QueryRow("SELECT id FROM users WHERE username=$1", username).Scan(&userId)
    if err != nil {
		return errors.New("User not found")
    }
	return nil
}

func CreatePost(w http.ResponseWriter, req *http.Request) {
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

	body := make([]byte, req.ContentLength)
	_, err = req.Body.Read(body)
	defer req.Body.Close()
	if err != io.EOF {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	postContent := PostContent{}
	err = json.Unmarshal(body, &postContent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &pb.CreatePostRequest{
		Username: username,
		Content:  postContent.Content,
	}
	
	resp, err := postServiceClient.CreatePost(context.Background(), grpcReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create post: %s", err.Error()), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(resp)
	w.WriteHeader(http.StatusOK)
}

func UpdatePost(w http.ResponseWriter, req *http.Request) {
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

	body := make([]byte, req.ContentLength)
	_, err = req.Body.Read(body)
	defer req.Body.Close()
	if err != io.EOF {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	postContent := PostContent{}
	err = json.Unmarshal(body, &postContent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	params := mux.Vars(req)
	postId, err := strconv.ParseUint(params["id"], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grpcReq := &pb.UpdatePostRequest{
		Id:		  postId,
		Username: username,
		Content:  postContent.Content,
	}
	
	_, err = postServiceClient.UpdatePost(context.Background(), grpcReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update post: %s", err.Error()), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeletePost(w http.ResponseWriter, req *http.Request) {
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

	grpcReq := &pb.DeletePostRequest{
		Id:		  postId,
		Username: username,
	}
	
	_, err = postServiceClient.DeletePost(context.Background(), grpcReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete post: %s", err.Error()), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func GetPost(w http.ResponseWriter, req *http.Request) {
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

	grpcReq := &pb.GetPostRequest{
		Id:		  postId,
		Username: username,
	}
	
	resp, err := postServiceClient.GetPost(context.Background(), grpcReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get post: %s", err.Error()), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(resp)
	w.WriteHeader(http.StatusOK)
}

func ListPosts(w http.ResponseWriter, req *http.Request) {
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

	fmt.Println(fmt.Sprintf("queryyyy %s", req.URL.Query()))

	limitStr := req.URL.Query().Get("limit")
	limit, err := strconv.ParseUint(limitStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid limit", http.StatusBadRequest)
		return
	}

	offsetStr := req.URL.Query().Get("offset")
	offset, err := strconv.ParseUint(offsetStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid offset", http.StatusBadRequest)
		return
	}

	grpcReq := &pb.ListPostsRequest{
		Limit:  limit,
		Offset: offset,
	}

	resp, err := postServiceClient.ListPosts(context.Background(), grpcReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list posts: %s", err.Error()), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(resp)
	w.WriteHeader(http.StatusOK)
}
