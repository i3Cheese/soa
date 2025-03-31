package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"proto/posts" // Import the generated gRPC package

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func setupPostsRoutes(router *gin.Engine) {
	postsServiceURL := os.Getenv("POSTS_URL")
	if postsServiceURL == "" {
		fmt.Println("POSTS_URL is required")
		os.Exit(1)
	}

	router.POST("/posts", func(c *gin.Context) {
		handleGRPCRequest(c, "CreatePost")
	})

	router.GET("/posts", func(c *gin.Context) {
		handleGRPCRequest(c, "GetPosts")
	})

	router.GET("/posts/:id", func(c *gin.Context) {
		handleGRPCRequest(c, "GetPostById")
	})

	router.PUT("/posts/:id", func(c *gin.Context) {
		handleGRPCRequest(c, "UpdatePost")
	})

	router.DELETE("/posts/:id", func(c *gin.Context) {
		handleGRPCRequest(c, "DeletePost")
	})
}

func handleGRPCRequest(c *gin.Context, method string) {
	grpcAddress := os.Getenv("POSTS_GRPC_URL")
	if grpcAddress == "" {
		fmt.Println("POSTS_GRPC_URL is required")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	conn, err := grpc.NewClient(grpcAddress)
	if err != nil {
		fmt.Printf("Failed to create gRPC client connection: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to service"})
		return
	}
	defer conn.Close()

	client := posts.NewPostServiceClient(conn)

	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil && method != "GetPostById" && method != "DeletePost" {
		fmt.Printf("Failed to parse JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// Prepare gRPC request based on the method
	var grpcResponse interface{}
	switch method {
	case "CreatePost":
		grpcResponse, err = client.CreatePost(context.Background(), &posts.CreatePostRequest{
			Title:       requestData["title"].(string),
			Description: requestData["description"].(string),
		})
	case "GetPosts":
		grpcResponse, err = client.GetPosts(context.Background(), &posts.GetPostsRequest{})
	case "GetPostById":
		grpcResponse, err = client.GetPostById(context.Background(), &posts.GetPostByIdRequest{
			Id: c.Param("id"),
		})
	case "UpdatePost":
		grpcResponse, err = client.UpdatePost(context.Background(), &posts.UpdatePostRequest{
			Id:          c.Param("id"),
			Title:       requestData["title"].(string),
			Description: requestData["description"].(string),
		})
	case "DeletePost":
		grpcResponse, err = client.DeletePost(context.Background(), &posts.DeletePostRequest{
			Id: c.Param("id"),
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid method"})
		return
	}

	if err != nil {
		fmt.Printf("gRPC request failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service error"})
		return
	}

	// Convert gRPC response to JSON
	responseData, err := json.Marshal(grpcResponse)
	if err != nil {
		fmt.Printf("Failed to marshal gRPC response: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process response"})
		return
	}

	c.Data(http.StatusOK, "application/json", responseData)
}
