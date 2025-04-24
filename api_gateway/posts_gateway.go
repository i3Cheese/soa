package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"msg.i3cheese.ru/proto/posts" // Import the generated gRPC package

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	router.DELETE("/posts/:id", func(c *gin.Context) {
		handleDeletePost(c, postsServiceURL)
	})
	router.PUT("/posts/:id", func(c *gin.Context) {
		handleUpdatePost(c, postsServiceURL)
	})
	router.GET("/posts/:id", func(c *gin.Context) {
		handleGetPostById(c, postsServiceURL)
	})
	router.GET("/posts", func(c *gin.Context) {
		handleGetPosts(c, postsServiceURL)
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

func handleDeletePost(c *gin.Context, postsServiceURL string) {
	client, ctx, closeConn, err := prepareRequest(c, postsServiceURL)
	if err != nil {
		return
	}
	defer closeConn()

	postId := c.Param("id")
	req := &posts.DeletePostRequest{PostId: postId}
	_, err = client.DeletePost(ctx, req)
	if err != nil {
		fmt.Printf("Failed to delete post: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete post"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func handleUpdatePost(c *gin.Context, postsServiceURL string) {
	client, ctx, closeConn, err := prepareRequest(c, postsServiceURL)
	if err != nil {
		return
	}
	defer closeConn()

	postId := c.Param("id")
	var reqBody CreatePostRequest
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		fmt.Printf("Failed to bind JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	req := &posts.UpdatePostRequest{
		PostId:      postId,
		Title:       reqBody.Title,
		Description: reqBody.Description,
		IsPrivate:   reqBody.IsPrivate,
	}
	resp, err := client.UpdatePost(ctx, req)
	if err != nil {
		fmt.Printf("Failed to update post: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update post"})
		return
	}
	c.JSON(http.StatusOK, resp.Post)
}

func handleGetPostById(c *gin.Context, postsServiceURL string) {
	client, ctx, closeConn, err := prepareRequest(c, postsServiceURL)
	if err != nil {
		return
	}
	defer closeConn()

	postId := c.Param("id")
	req := &posts.GetPostByIdRequest{PostId: postId}
	resp, err := client.GetPostById(ctx, req)
	if err != nil {
		fmt.Printf("Failed to fetch post: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch post"})
		return
	}
	c.JSON(http.StatusOK, resp.Post)
}

func handleGetPosts(c *gin.Context, postsServiceURL string) {
	client, ctx, closeConn, err := prepareRequest(c, postsServiceURL)
	if err != nil {
		return
	}
	defer closeConn()

	startFrom := c.Query("start_from")
	limit := c.Query("limit")
	if limit == "" {
		limit = "10" // Default limit
	}
	req := &posts.GetPostsRequest{}
	if startFrom != "" {
		parsedTime, err := time.Parse(time.RFC3339, startFrom)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_from format"})
			return
		}
		req.StartFrom = timestamppb.New(parsedTime)
	}
	parsedLimit, err := strconv.Atoi(limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit format"})
		return
	}
	req.Limit = int32(parsedLimit)

	resp, err := client.GetPosts(ctx, req)
	if err != nil {
		fmt.Printf("Failed to fetch posts: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}

	// Ensure posts is an empty array if nil
	if resp.Posts == nil {
		resp.Posts = []*posts.Post{}
	}

	c.JSON(http.StatusOK, gin.H{"posts": resp.Posts, "total_count": resp.TotalCount})
}
