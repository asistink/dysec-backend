package main

import (
	"Dysec/internal/ai"
	"Dysec/internal/database"
	"Dysec/internal/handlers"
	"Dysec/internal/middleware"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		log.Println("Could not find config.yaml, using environment variables only.")
	}
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Could not connect to the database: %v", err)
	}

	jwtSecret := viper.GetString("jwt.secret_key")
	if jwtSecret == "" {
		log.Fatal("JWT secret key not found in config")
	}
	geminiAPIKey := viper.GetString("gemini.api_key")
	if geminiAPIKey == "" {
		log.Fatal("Gemini API key not found in config")
	}
	aiService, err := ai.NewService(geminiAPIKey)
	if err != nil {
		log.Fatalf("Could not initialize AI service: %v", err)
	}
	h := handlers.New(db, jwtSecret, aiService)

	router := gin.Default()
	v1 := router.Group("/api/v1")
	{
		v1.POST("/auth/google", h.GoogleAuthHandler)

		authorized := v1.Group("/")
		authorized.Use(middleware.JWTMiddleware(jwtSecret))
		{
			authorized.POST("/tests/start", h.StartSessionHandler)
			authorized.POST("/tests/:id/submit", h.SubmitTestHandler)
			authorized.GET("/tests/history", h.TestHistoryHandler)
			authorized.GET("/users/me", h.UserProfileHandler)
		}
	}

	log.Println("Starting server on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}

}
