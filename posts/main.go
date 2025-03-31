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

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen on port 50051: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("gRPC server is running on port 50051")
	if err := grpcServer.Serve(listener); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to serve gRPC server: %v\n", err)
		os.Exit(1)
	}
}
