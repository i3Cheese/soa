package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	passportServiceURL := os.Getenv("PASSPORT_URL")
	if passportServiceURL == "" {
		fmt.Println("PASSPORT_URL is required")
		os.Exit(1)
	}

	router.POST("/register", func(c *gin.Context) {
		proxyRequest(c, passportServiceURL+"/register")
	})

	router.POST("/login", func(c *gin.Context) {
		proxyRequest(c, passportServiceURL+"/login")
	})

	router.Run(fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT")))
}

func proxyRequest(c *gin.Context, url string) {
	req, err := http.NewRequest(c.Request.Method, url, c.Request.Body)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	req.Header = c.Request.Header

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to proxy request: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to proxy request"})
		return
	}
	defer resp.Body.Close()

	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}
