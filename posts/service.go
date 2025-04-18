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
		fmt.Printf("Failed to get metadata from context\n")
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
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		fmt.Printf("Failed to get metadata from context\n")
		return nil, fmt.Errorf("failed to get metadata from context")
	}
	actorUserId := md.Get("actor_user_id")[0]

	query := `SELECT creator_id FROM posts WHERE post_id = $1`
	row := s.App.DB.QueryRow(ctx, query, req.PostId)

	var creatorId string
	err := row.Scan(&creatorId)
	if err != nil {
		fmt.Printf("Failed to fetch post: %v\n", err)
		return nil, fmt.Errorf("failed to fetch post: %v", err)
	}
	if actorUserId != creatorId {
		fmt.Printf("Unauthorized: actor does not match creator\n")
		return nil, fmt.Errorf("unauthorized: actor does not match creator")
	}

	// Implement logic to delete a post from the database
	query = `DELETE FROM posts WHERE post_id = $1`
	_, err = s.App.DB.Exec(ctx, query, req.PostId)
	if err != nil {
		fmt.Printf("Failed to delete post: %v\n", err)
		return nil, fmt.Errorf("failed to delete post: %v", err)
	}

	return &posts.DeletePostResponse{Success: true}, nil
}

func (s *PostServiceServer) UpdatePost(ctx context.Context, req *posts.UpdatePostRequest) (*posts.UpdatePostResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		fmt.Printf("Failed to get metadata from context\n")
		return nil, fmt.Errorf("failed to get metadata from context")
	}
	actorUserId := md.Get("actor_user_id")[0]

	query := `SELECT creator_id FROM posts WHERE post_id = $1`
	row := s.App.DB.QueryRow(ctx, query, req.PostId)

	var creatorId string
	err := row.Scan(&creatorId)
	if err != nil {
		fmt.Printf("Failed to fetch post: %v\n", err)
		return nil, fmt.Errorf("failed to fetch post: %v", err)
	}
	if actorUserId != creatorId {
		fmt.Printf("Unauthorized: actor does not match creator\n")
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

	var updatedAt time.Time
	err = row.Scan(&updatedAt)
	if err != nil {
		fmt.Printf("Failed to update post: %v\n", err)
		return nil, fmt.Errorf("failed to update post: %v", err)
	}
	post.UpdatedAt = timestamppb.New(updatedAt)

	return &posts.UpdatePostResponse{Post: &post}, nil
}

func (s *PostServiceServer) GetPostById(ctx context.Context, req *posts.GetPostByIdRequest) (*posts.GetPostByIdResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		fmt.Printf("Failed to get metadata from context\n")
		return nil, fmt.Errorf("failed to get metadata from context")
	}
	actorUserId := md.Get("actor_user_id")[0]

	query := `SELECT creator_id, is_private FROM posts WHERE post_id = $1`
	row := s.App.DB.QueryRow(ctx, query, req.PostId)

	var creatorId string
	var isPrivate bool
	err := row.Scan(&creatorId, &isPrivate)
	if err != nil {
		fmt.Printf("Failed to fetch post: %v\n", err)
		return nil, fmt.Errorf("failed to fetch post: %v", err)
	}
	if isPrivate && actorUserId != creatorId {
		fmt.Printf("Unauthorized: actor does not have access to private post\n")
		return nil, fmt.Errorf("unauthorized: actor does not have access to private post")
	}

	// Implement logic to fetch a post by ID from the database
	query = `SELECT post_id, title, description, creator_id, created_at, updated_at, is_private FROM posts WHERE post_id = $1`
	row = s.App.DB.QueryRow(ctx, query, req.PostId)

	var post posts.Post
	var createdAt, updatedAt time.Time
	err = row.Scan(&post.PostId, &post.Title, &post.Description, &post.CreatorId, &createdAt, &updatedAt, &post.IsPrivate)
	if err != nil {
		fmt.Printf("Failed to fetch post by ID: %v\n", err)
		return nil, fmt.Errorf("failed to fetch post by ID: %v", err)
	}
	post.CreatedAt = timestamppb.New(createdAt)
	post.UpdatedAt = timestamppb.New(updatedAt)

	return &posts.GetPostByIdResponse{Post: &post}, nil
}

func (s *PostServiceServer) GetPosts(ctx context.Context, req *posts.GetPostsRequest) (*posts.GetPostsResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		fmt.Printf("Failed to get metadata from context\n")
		return nil, fmt.Errorf("failed to get metadata from context")
	}
	actorUserId := md.Get("actor_user_id")[0]
	query := `SELECT post_id, title, description, creator_id, created_at, updated_at, is_private 
			  FROM posts 
			  ORDER BY created_at ASC 
			  LIMIT $1`
	startFrom := time.Unix(0, 0)
	if req.StartFrom != nil {
		startFrom = req.StartFrom.AsTime()
	}
	_, _ = actorUserId, startFrom
	rows, err := s.App.DB.Query(ctx, query, req.Limit)
	if err != nil {
		fmt.Printf("Failed to fetch posts: %v\n", err)
		return nil, fmt.Errorf("failed to fetch posts: %v", err)
	}
	defer rows.Close()

	var postsList []*posts.Post
	for rows.Next() {
		var post posts.Post
		var createdAt, updatedAt time.Time
		err := rows.Scan(&post.PostId, &post.Title, &post.Description, &post.CreatorId, &createdAt, &updatedAt, &post.IsPrivate)
		if err != nil {
			fmt.Printf("Failed to scan post: %v\n", err)
			return nil, fmt.Errorf("failed to scan post: %v", err)
		}
		post.CreatedAt = timestamppb.New(createdAt)
		post.UpdatedAt = timestamppb.New(updatedAt)
		postsList = append(postsList, &post)
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("Error iterating over rows: %v\n", err)
		return nil, fmt.Errorf("error iterating over rows: %v", err)
	}

	// len of postsList
	var totalCount int32 = int32(len(postsList))

	return &posts.GetPostsResponse{Posts: postsList, TotalCount: totalCount}, nil
}
