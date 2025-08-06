package middleware

import (
	"Dysec/internal/models"
	"context"
	//"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
)

// GoogleTokenMiddleware akan memvalidasi Google ID Token di setiap request
func GoogleTokenMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}
		tokenString := parts[1]

		// Validasi token ke server Google
		payload, err := idtoken.Validate(context.Background(), tokenString, "")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Google ID Token"})
			return
		}

		// Cari user di database kita berdasarkan Google ID dari token
		var user models.User
		googleID := payload.Subject
		if err := db.Where("google_id = ?", googleID).First(&user).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not found for this token"})
			return
		}

		// Simpan user_id internal kita ke dalam context untuk digunakan handler selanjutnya
		c.Set("user_id", float64(user.ID)) // Gin context lebih aman dengan float64 untuk angka
		c.Next()
	}
}
