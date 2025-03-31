package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	router.POST("/passport/register", func(c *gin.Context) {
		proxyRequest(c, passportServiceURL+"/register", false)
	})

	router.POST("/passport/login", func(c *gin.Context) {
		proxyRequest(c, passportServiceURL+"/login", false)
	})
	router.GET("/passport/me", func(c *gin.Context) {
		proxyRequest(c, passportServiceURL+"/me", true)
	})
	router.PUT("/passport/me", func(c *gin.Context) {
		proxyRequest(c, passportServiceURL+"/me", true)
	})

	setupPostsRoutes(router)

	router.Run(fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT")))
}

func CheckToken(token string) (user_id string, err error) {
	body, err := json.Marshal(map[string]any{"Token": token})
	if err != nil {
		fmt.Printf("Failed to marshal JSON: %v\n", err)
		return "", err
	}

	req, err := http.NewRequest("GET", os.Getenv("PASSPORT_URL")+"/check_token", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		return "", err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to send request: %v\n", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Failed to check token: %v\n", resp.Status)
		return "", fmt.Errorf("failed to check token: %v", resp.Status)
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response body: %v\n", err)
		return
	}
	var response map[string]any = make(map[string]any)
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Printf("Failed to unmarshal JSON: %v\n", err)
		return
	}
	user_id = response["user_id"].(string)
	fmt.Printf("User ID: %s\n", user_id)

	return user_id, nil
}

func proxyRequest(c *gin.Context, url string, authRequired bool) {
	req, err := http.NewRequest(c.Request.Method, url, c.Request.Body)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	req.Header = c.Request.Header
	if authRequired {
		token := c.Request.Header.Get("Authorization")
		if token == "" {
			fmt.Println("Authorization token is required")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token is required"})
			return
		}
		user_id, err := CheckToken(token)
		if err != nil {
			fmt.Printf("Failed to check token: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to check token"})
			return
		}
		req.Header.Set("X-User-Id", user_id)

		req.Header.Del("Authorization")
	}

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
