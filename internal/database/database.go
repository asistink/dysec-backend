package database

import (
	"Dysec/internal/models"
	"errors"
	"fmt"
	"log"

	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect() (*gorm.DB, error) {
	// Ambil nilai dari Viper. Viper akan otomatis memilih
	// dari Environment Variable (jika ada) atau dari config.yaml.
	host := viper.GetString("DB_HOST")
	port := viper.GetString("DB_PORT")
	user := viper.GetString("DB_USER")
	password := viper.GetString("DB_PASSWORD")
	dbname := viper.GetString("DB_NAME")

	// Fallback ke config.yaml jika env var kosong (untuk development lokal)
	if host == "" {
		host = viper.GetString("database.host")
		port = viper.GetString("database.port")
		user = viper.GetString("database.user")
		password = viper.GetString("database.password")
		dbname = viper.GetString("database.dbname")
	}

	// Validasi
	if host == "" || port == "" || user == "" || dbname == "" {
		return nil, errors.New("database configuration is incomplete")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		host, user, password, dbname, port,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection established")

	err = db.AutoMigrate(&models.User{}, &models.UserTest{}, &models.AiScore{}, &models.Question{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database migrated successfully")

	return db, nil
}
