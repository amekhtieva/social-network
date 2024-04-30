package main

import (
	"context"
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	"gorm.io/gorm"

	pb "post_service/proto"
)

type Server struct {
	DB *gorm.DB
	pb.UnimplementedPostServiceServer
}

type Post struct {
	Id       uint64 `gorm:"primarykey"`
	Username string
	Content  string
}

func (s *Server) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.CreatePostResponse, error) {
	post := &Post{
		Username:  req.Username,
		Content: req.Content,
	}
	s.DB.Create(post)

	return &pb.CreatePostResponse{
		PostId: post.Id,
	}, nil
}

func (s *Server) UpdatePost(ctx context.Context, req *pb.UpdatePostRequest) (*empty.Empty, error) {
	post := &Post{}
	err := s.DB.First(&post, req.Id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("Post not found")
		} else {
			return nil, err
		}
	}

	if post.Username != req.Username {
		return nil, errors.New("Only the creator can update the post")
	}

	post.Content = req.Content
	s.DB.Save(&post)

	return &empty.Empty{}, nil
}

func (s *Server) DeletePost(ctx context.Context, req *pb.DeletePostRequest) (*empty.Empty, error) {
	post := &Post{}
	err := s.DB.First(&post, req.Id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("Post not found")
		} else {
			return nil, err
		}
	}

	if post.Username != req.Username {
		return nil, errors.New("Only the creator can delete the post")
	}

	s.DB.Delete(&Post{}, req.Id)
	return &empty.Empty{}, nil
}

func (s *Server) GetPost(ctx context.Context, req *pb.GetPostRequest) (*pb.GetPostResponse, error) {
	post := &Post{}
	err := s.DB.First(&post, req.Id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("Post not found")
		} else {
			return nil, err
		}
	}

	if post.Username != req.Username {
		return nil, errors.New("Only the creator can get the post")
	}

	return &pb.GetPostResponse{
		Post: &pb.Post{
			Id:      post.Id,
			Username:  post.Username,
			Content: post.Content,
		},
	}, nil
}

func (s *Server) ListPosts(ctx context.Context, req *pb.ListPostsRequest) (*pb.ListPostsResponse, error) {
	var posts []*Post
	s.DB.Limit(int(req.Limit)).Offset(int(req.Offset)).Find(&posts)

	var postsPb []*pb.Post
	for _, post := range posts {
		postsPb = append(postsPb, &pb.Post{
			Id:      post.Id,
			Username:  post.Username,
			Content: post.Content,
		})
	}

	return &pb.ListPostsResponse{
		Posts: postsPb,
	}, nil
}
