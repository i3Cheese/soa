package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"msg.i3cheese.ru/proto/posts" // Import the generated gRPC package

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func setupPostsRoutes(router *gin.Engine) {
	postsServiceURL := os.Getenv("POSTS_URL")
	if postsServiceURL == "" {
		fmt.Println("POSTS_URL is required")
		os.Exit(1)
	}

	router.POST("/posts", func(c *gin.Context) {
		handleCreatePost(c, postsServiceURL)
	})
}

type CreatePostRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
}

type Post struct {
	PostId      string    `json:"post_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatorId   string    `json:"creator_id"`
	IsPrivate   bool      `json:"is_private"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Retunrn client, context, func to defer and close the connection
// and error if any
func prepareRequest(c *gin.Context, postsServiceURL string) (posts.PostServiceClient, context.Context, func(), error) {
	user_id, err := CheckToken(c.Request.Header.Get("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return nil, nil, nil, err
	}
	conn, err := grpc.NewClient(postsServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to posts service"})
		return nil, nil, nil, err
	}
	client := posts.NewPostServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	md := metadata.Pairs("actor_user_id", user_id)
	ctx = metadata.NewOutgoingContext(ctx, md)
	return client, ctx, func() {
		cancel()
		conn.Close()
	}, nil
}

func handleCreatePost(c *gin.Context, postsServiceURL string) {
	client, ctx, closeConn, err := prepareRequest(c, postsServiceURL)
	if err != nil {
		return
	}
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("Failed to bind JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	defer closeConn()
	createPostRequest := &posts.CreatePostRequest{
		Title:       req.Title,
		Description: req.Description,
		IsPrivate:   req.IsPrivate,
	}
	fmt.Printf("ctx User ID: %s\n", ctx.Value("actor_user_id"))
	resp, err := client.CreatePost(ctx, createPostRequest)
	if err != nil {
		fmt.Printf("Failed to create post: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
		return
	}
	// parse the response into the Post struct
	post := Post{
		PostId:      resp.Post.PostId,
		Title:       resp.Post.Title,
		Description: resp.Post.Description,
		CreatorId:   resp.Post.CreatorId,
		IsPrivate:   resp.Post.IsPrivate,
		CreatedAt:   resp.Post.CreatedAt.AsTime(),
		UpdatedAt:   resp.Post.UpdatedAt.AsTime(),
	}
	c.JSON(http.StatusCreated, post)
}
