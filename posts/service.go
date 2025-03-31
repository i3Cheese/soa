package main

import (
	"context"
	"fmt"

	"msg.i3cheese.ru/proto/posts"
)

type PostServiceServer struct {
	posts.UnimplementedPostServiceServer
	App *App
}

func (s *PostServiceServer) CreatePost(ctx context.Context, req *posts.CreatePostRequest) (*posts.CreatePostResponse, error) {
	// Implement logic to create a post in the database
	query := `INSERT INTO posts (title, description, creator_id, is_private) VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`
	row := s.App.DB.QueryRow(ctx, query, req.Title, req.Description, req.CreatorId, req.IsPrivate)

	var post posts.Post
	post.Title = req.Title
	post.Description = req.Description
	post.CreatorId = req.CreatorId
	post.IsPrivate = req.IsPrivate

	err := row.Scan(&post.Id, &post.CreatedAt, &post.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create post: %v", err)
	}

	return &posts.CreatePostResponse{Post: &post}, nil
}

func (s *PostServiceServer) DeletePost(ctx context.Context, req *posts.DeletePostRequest) (*posts.DeletePostResponse, error) {
	// Implement logic to delete a post from the database
	query := `DELETE FROM posts WHERE id = $1`
	_, err := s.App.DB.Exec(ctx, query, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete post: %v", err)
	}

	return &posts.DeletePostResponse{Success: true}, nil
}

func (s *PostServiceServer) UpdatePost(ctx context.Context, req *posts.UpdatePostRequest) (*posts.UpdatePostResponse, error) {
	// Implement logic to update a post in the database
	query := `UPDATE posts SET title = $1, description = $2, is_private = $3 WHERE id = $4 RETURNING updated_at`
	row := s.App.DB.QueryRow(ctx, query, req.Title, req.Description, req.IsPrivate, req.Id)

	var post posts.Post
	post.Id = req.Id
	post.Title = req.Title
	post.Description = req.Description
	post.IsPrivate = req.IsPrivate

	err := row.Scan(&post.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update post: %v", err)
	}

	return &posts.UpdatePostResponse{Post: &post}, nil
}

func (s *PostServiceServer) GetPostById(ctx context.Context, req *posts.GetPostByIdRequest) (*posts.GetPostByIdResponse, error) {
	// Implement logic to fetch a post by ID from the database
	query := `SELECT id, title, description, creator_id, created_at, updated_at, is_private FROM posts WHERE id = $1`
	row := s.App.DB.QueryRow(ctx, query, req.Id)

	var post posts.Post
	err := row.Scan(&post.Id, &post.Title, &post.Description, &post.CreatorId, &post.CreatedAt, &post.UpdatedAt, &post.IsPrivate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch post by ID: %v", err)
	}

	return &posts.GetPostByIdResponse{Post: &post}, nil
}

func (s *PostServiceServer) GetPosts(ctx context.Context, req *posts.GetPostsRequest) (*posts.GetPostsResponse, error) {
	// Implement logic to fetch posts with pagination
	query := `SELECT id, title, description, creator_id, created_at, updated_at, is_private FROM posts LIMIT $1 OFFSET $2`
	rows, err := s.App.DB.Query(ctx, query, req.PageSize, (req.Page-1)*req.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posts: %v", err)
	}
	defer rows.Close()

	var postsList []*posts.Post
	for rows.Next() {
		var post posts.Post
		err := rows.Scan(&post.Id, &post.Title, &post.Description, &post.CreatorId, &post.CreatedAt, &post.UpdatedAt, &post.IsPrivate)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %v", err)
		}
		postsList = append(postsList, &post)
	}

	return &posts.GetPostsResponse{Posts: postsList, TotalCount: int32(len(postsList))}, nil
}
