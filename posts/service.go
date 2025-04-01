package main

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	"msg.i3cheese.ru/proto/posts"
)

type PostServiceServer struct {
	posts.UnimplementedPostServiceServer
	App *App
}

func (s *PostServiceServer) CreatePost(ctx context.Context, req *posts.CreatePostRequest) (*posts.CreatePostResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get metadata from context")
	}
	actorUserId := md.Get("actor_user_id")[0]

	// Implement logic to create a post in the database
	query := `INSERT INTO posts (title, description, creator_id, is_private) VALUES ($1, $2, $3, $4) RETURNING post_id, created_at, updated_at`
	row := s.App.DB.QueryRow(ctx, query, req.Title, req.Description, actorUserId, req.IsPrivate)

	var post posts.Post
	post.Title = req.Title
	post.Description = req.Description
	post.CreatorId = actorUserId
	post.IsPrivate = req.IsPrivate

	var createdAt, updatedAt time.Time
	err := row.Scan(&post.PostId, &createdAt, &updatedAt)
	if err == nil {
		post.CreatedAt = timestamppb.New(createdAt)
		post.UpdatedAt = timestamppb.New(updatedAt)
	}
	if err != nil {
		fmt.Printf("Failed to create post: %v\n", err)
		return nil, fmt.Errorf("failed to create post: %v", err)
	}

	return &posts.CreatePostResponse{Post: &post}, nil
}

func (s *PostServiceServer) DeletePost(ctx context.Context, req *posts.DeletePostRequest) (*posts.DeletePostResponse, error) {
	actorUserId := ctx.Value("actor_user_id").(string)
	query := `SELECT creator_id FROM posts WHERE post_id = $1`
	row := s.App.DB.QueryRow(ctx, query, req.PostId)

	var creatorId string
	err := row.Scan(&creatorId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch post: %v", err)
	}
	if actorUserId != creatorId {
		return nil, fmt.Errorf("unauthorized: actor does not match creator")
	}

	// Implement logic to delete a post from the database
	query = `DELETE FROM posts WHERE post_id = $1`
	_, err = s.App.DB.Exec(ctx, query, req.PostId)
	if err != nil {
		return nil, fmt.Errorf("failed to delete post: %v", err)
	}

	return &posts.DeletePostResponse{Success: true}, nil
}

func (s *PostServiceServer) UpdatePost(ctx context.Context, req *posts.UpdatePostRequest) (*posts.UpdatePostResponse, error) {
	actorUserId := ctx.Value("actor_user_id").(string)
	query := `SELECT creator_id FROM posts WHERE post_id = $1`
	row := s.App.DB.QueryRow(ctx, query, req.PostId)

	var creatorId string
	err := row.Scan(&creatorId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch post: %v", err)
	}
	if actorUserId != creatorId {
		return nil, fmt.Errorf("unauthorized: actor does not match creator")
	}

	// Implement logic to update a post in the database
	query = `UPDATE posts SET title = $1, description = $2, is_private = $3 WHERE post_id = $4 RETURNING updated_at`
	row = s.App.DB.QueryRow(ctx, query, req.Title, req.Description, req.IsPrivate, req.PostId)

	var post posts.Post
	post.PostId = req.PostId
	post.Title = req.Title
	post.Description = req.Description
	post.IsPrivate = req.IsPrivate

	err = row.Scan(&post.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update post: %v", err)
	}

	return &posts.UpdatePostResponse{Post: &post}, nil
}

func (s *PostServiceServer) GetPostById(ctx context.Context, req *posts.GetPostByIdRequest) (*posts.GetPostByIdResponse, error) {
	actorUserId := ctx.Value("actor_user_id").(string)
	query := `SELECT creator_id, is_private FROM posts WHERE post_id = $1`
	row := s.App.DB.QueryRow(ctx, query, req.PostId)

	var creatorId string
	var isPrivate bool
	err := row.Scan(&creatorId, &isPrivate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch post: %v", err)
	}
	if isPrivate && actorUserId != creatorId {
		return nil, fmt.Errorf("unauthorized: actor does not have access to private post")
	}

	// Implement logic to fetch a post by ID from the database
	query = `SELECT post_id, title, description, creator_id, created_at, updated_at, is_private FROM posts WHERE post_id = $1`
	row = s.App.DB.QueryRow(ctx, query, req.PostId)

	var post posts.Post
	err = row.Scan(&post.PostId, &post.Title, &post.Description, &post.CreatorId, &post.CreatedAt, &post.UpdatedAt, &post.IsPrivate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch post by ID: %v", err)
	}

	return &posts.GetPostByIdResponse{Post: &post}, nil
}

func (s *PostServiceServer) GetPosts(ctx context.Context, req *posts.GetPostsRequest) (*posts.GetPostsResponse, error) {
	actorUserId := ctx.Value("actor_user_id").(string)
	query := `SELECT post_id, title, description, creator_id, created_at, updated_at, is_private FROM posts`
	rows, err := s.App.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posts: %v", err)
	}
	defer rows.Close()

	var postsList []*posts.Post
	for rows.Next() {
		var post posts.Post
		err := rows.Scan(&post.PostId, &post.Title, &post.Description, &post.CreatorId, &post.CreatedAt, &post.UpdatedAt, &post.IsPrivate)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %v", err)
		}
		if !post.IsPrivate || post.CreatorId == actorUserId {
			postsList = append(postsList, &post)
		}
	}

	return &posts.GetPostsResponse{Posts: postsList, TotalCount: int32(len(postsList))}, nil
}
