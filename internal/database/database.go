package database

import (
	"Dysec/internal/models"
	"errors"
	"fmt"
	"log"
	"strings" // <-- IMPORT BARU

	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect() (*gorm.DB, error) {
	// --- PERBAIKAN LOGIKA VIPER ---

	// 1. Atur nama dan path file konfigurasi
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	// 2. Atur agar bisa membaca dari environment variables juga (untuk Docker)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 3. Coba baca file konfigurasi
	if err := viper.ReadInConfig(); err != nil {
		log.Println("Could not find config.yaml, will rely on environment variables.")
	}

	// 4. Ambil nilai dengan path lengkap dari file YAML
	host := viper.GetString("database.host")
	port := viper.GetString("database.port")
	user := viper.GetString("database.user")
	password := viper.GetString("database.password")
	dbname := viper.GetString("database.dbname")

	// 5. Validasi: Pastikan nilai tidak kosong
	if host == "" || port == "" || user == "" || dbname == "" {
		return nil, errors.New("database configuration is incomplete. Please check config.yaml or environment variables")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		host, user, password, dbname, port,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection established")

	log.Println("Running database migrations...")
	err = db.AutoMigrate(&models.User{}, &models.UserTest{}, &models.AiScore{}, &models.Question{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database migrated successfully")

	return db, nil
}
