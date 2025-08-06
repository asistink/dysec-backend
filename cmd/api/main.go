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
	// --- PENGATURAN KONFIGURASI DI PALING ATAS ---
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs") // Path ke folder config
	viper.AutomaticEnv()             // Memungkinkan pembacaan dari environment variable
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Baca file konfigurasi
	if err := viper.ReadInConfig(); err != nil {
		// Jangan berhenti jika file tidak ada, karena mungkin kita pakai env var di Docker
		log.Printf("Warning: Could not read config.yaml: %v. Relying on environment variables.", err)
	}

	// 1. Inisialisasi Database
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Could not connect to the database: %v", err)
	}

	// 2. Baca Konfigurasi Gemini API Key
	geminiAPIKey := viper.GetString("GEMINI_API_KEY") // Coba dari env var
	if geminiAPIKey == "" {
		geminiAPIKey = viper.GetString("gemini.api_key") // Fallback ke config.yaml
	}
	if geminiAPIKey == "" {
		log.Fatal("Gemini API key not found in config or environment variables")
	}

	// 3. Inisialisasi AI Service
	aiService, err := ai.NewService(geminiAPIKey)
	if err != nil {
		log.Fatalf("Could not initialize AI service: %v", err)
	}

	// 4. Inisialisasi Handler
	h := handlers.New(db, aiService)

	// 5. Setup Router
	router := gin.Default()
	v1 := router.Group("/api/v1")
	{
		v1.POST("/auth/google", h.GoogleAuthHandler)

		authorized := v1.Group("/")
		authorized.Use(middleware.GoogleTokenMiddleware(db))
		{
			authorized.POST("/tests/start", h.StartSessionHandler)
			authorized.POST("/tests/:id/submit", h.SubmitTestHandler)
			authorized.GET("/tests/history", h.TestHistoryHandler)
			authorized.GET("/users/me", h.UserProfileHandler)
		}
	}

	// 6. Jalankan Server
	log.Println("Starting server on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
