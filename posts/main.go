package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"msg.i3cheese.ru/proto/posts"
)

type App struct {
	DB          *pgx.Conn
	KafkaWriter *kafka.Writer
}

func connectWithRetries(ctx context.Context, dsn string, maxRetries int) (*pgx.Conn, error) {
	var conn *pgx.Conn
	var err error
	for i := 0; i < maxRetries; i++ {
		conn, err = pgx.Connect(ctx, dsn)
		if err == nil {
			return conn, nil
		}
		fmt.Fprintf(os.Stderr, "Attempt %d: Unable to connect to database: %v\n", i+1, err)
		time.Sleep(2 * time.Second) // Add a delay between retries
	}
	return nil, err
}

func main() {
	conn, err := connectWithRetries(context.Background(), os.Getenv("DATABASE_URL"), 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database after retries: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "kafka:9092"
	}
	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokers),
		Balancer: &kafka.LeastBytes{},
	}

	app := &App{DB: conn, KafkaWriter: kafkaWriter}

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
