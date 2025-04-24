package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/segmentio/kafka-go"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	DB          *pgx.Conn
	KafkaWriter *kafka.Writer
}

type RegisterRequest struct {
	Login       string `json:"login"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Name        string `json:"name"`
	Surname     string `json:"surname"`
	DateOfBirth string `json:"date_of_birth"`
	PhoneNumber string `json:"phone_number"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type UpdateUserRequest struct {
	Email       string `json:"email"`
	Name        string `json:"name"`
	Surname     string `json:"surname"`
	DateOfBirth string `json:"date_of_birth"`
	PhoneNumber string `json:"phone_number"`
}

func (app *App) Register(c *gin.Context) {
	var user RegisterRequest
	if err := c.ShouldBindJSON(&user); err != nil {
		fmt.Printf("Failed to bind JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Failed to hash password: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	now := time.Now()
	var userID string
	err = app.DB.QueryRow(
		context.Background(),
		"INSERT INTO users (login, email, hashed_password, name, surname, date_of_birth, phone_number, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING user_id",
		user.Login, user.Email, string(hashedPassword), user.Name, user.Surname, user.DateOfBirth, user.PhoneNumber, now, now,
	).Scan(&userID)
	if err != nil {
		fmt.Printf("Failed to register user: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	// Send registration event to Kafka
	if app.KafkaWriter != nil {
		event := fmt.Sprintf(`{"user_id":"%s","registered_at":"%s"}`, userID, now.Format(time.RFC3339))
		err := app.KafkaWriter.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(userID),
				Value: []byte(event),
			},
		)
		if err != nil {
			fmt.Printf("Failed to send registration event to Kafka: %v\n", err)
			// Do not fail registration on Kafka error
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "User registered successfully"})
}

func (app *App) Login(c *gin.Context) {
	var loginReq LoginRequest
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		fmt.Printf("Failed to bind JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var hashedPassword string
	var user_id string
	err := app.DB.QueryRow(context.Background(), "SELECT hashed_password, user_id FROM users WHERE login=$1", loginReq.Login).Scan(&hashedPassword, &user_id)
	if err != nil {
		fmt.Printf("Failed to find user: %v\n", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid login"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(loginReq.Password))
	if err != nil {
		fmt.Printf("Failed to compare passwords: %v\n", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user_id,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		fmt.Printf("Failed to generate token: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString, "user_id": user_id})
}

type CheckTokenRequest struct {
	Token string `json:"token"`
}

func (app *App) CheckToken(c *gin.Context) {
	var tokenReq CheckTokenRequest
	if err := c.ShouldBindJSON(&tokenReq); err != nil {
		fmt.Printf("Failed to bind JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	token, err := jwt.Parse(tokenReq.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		fmt.Printf("Failed to parse token: %v\n", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		fmt.Printf("Failed to validate token: %v\n", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}
	fmt.Printf("claims: %v\n", claims)

	c.JSON(http.StatusOK, gin.H{"user_id": claims["user_id"]})
}

type UserInfo struct {
	Login       string `json:"login"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Surname     string `json:"surname"`
	DateOfBirth string `json:"date_of_birth"`
	PhoneNumber string `json:"phone_number"`
}

func (app *App) GetMyInfo(c *gin.Context) {
	// Get user ID from header
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		fmt.Println("Failed to get user ID from header")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user UserInfo
	var date_of_birth pgtype.Date
	err := app.DB.QueryRow(context.Background(), "SELECT login, email, name, surname, date_of_birth, phone_number FROM users WHERE user_id=$1", userID).Scan(&user.Login, &user.Email, &user.Name, &user.Surname, &date_of_birth, &user.PhoneNumber)
	user.DateOfBirth = date_of_birth.Time.Format("2006-01-02")
	if err != nil {
		fmt.Printf("Failed to find user with id %s: %v\n", userID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (app *App) UpdateMyInfo(c *gin.Context) {
	// Get user ID from header
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		fmt.Println("Failed to get user ID from header")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var user UpdateUserRequest
	if err := c.ShouldBindJSON(&user); err != nil {
		fmt.Printf("Failed to bind JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	_, err := app.DB.Exec(
		context.Background(),
		"UPDATE users SET email=$1, name=$2, surname=$3, date_of_birth=$4, phone_number=$5, updated_at=$6 WHERE user_id=$7",
		user.Email, user.Name, user.Surname, user.DateOfBirth, user.PhoneNumber, time.Now(), userID,
	)
	if err != nil {
		fmt.Printf("Failed to update user: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "User updated successfully"})
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

	// Kafka writer setup
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "kafka:9092"
	}
	kafkaTopic := os.Getenv("KAFKA_REGISTRATION_TOPIC")
	if kafkaTopic == "" {
		kafkaTopic = "user_registrations"
	}
	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBrokers),
		Topic:    kafkaTopic,
		Balancer: &kafka.LeastBytes{},
	}

	app := &App{DB: conn, KafkaWriter: kafkaWriter}

	router := gin.Default()
	router.POST("/register", app.Register)
	router.POST("/login", app.Login)
	router.GET("/check_token", app.CheckToken)
	router.GET("/me", app.GetMyInfo)
	router.PUT("/me", app.UpdateMyInfo)

	router.Run(fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT")))
}
