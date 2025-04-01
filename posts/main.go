package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"msg.i3cheese.ru/proto/posts"
)

type App struct {
	DB *pgx.Conn
}

func main() {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	app := &App{DB: conn}

	grpcServer := grpc.NewServer()
	postService := &PostServiceServer{App: app}
	posts.RegisterPostServiceServer(grpcServer, postService)

	// Enable reflection for debugging
	reflection.Register(grpcServer)
	port := os.Getenv("POSTS_PORT")
	if port == "" {
		fmt.Println("POSTS_PORT is required")
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen on port %s: %v\n", port, err)
		os.Exit(1)
	}

	fmt.Printf("gRPC server is running on port %s\n", port)
	if err := grpcServer.Serve(listener); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to serve gRPC server: %v\n", err)
		os.Exit(1)
	}
}
