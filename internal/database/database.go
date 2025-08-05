package database

import (
	"Dysec/internal/models"
	"fmt"
	"log"
	"strings" // <-- IMPORT BARU

	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect() (*gorm.DB, error) {
	// Prioritaskan Environment Variables
	viper.SetEnvPrefix("DB") // Hanya cari env var yang diawali DB_
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Baca juga dari file config sebagai fallback/default
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	if err := viper.ReadInConfig(); err != nil {
		log.Println("Could not find config.yaml, using environment variables only.")
	}

	// Ambil nilai. Viper akan otomatis mengambil dari Env Var jika ada,
	// jika tidak ada, ia akan ambil dari file config.
	host := viper.GetString("HOST")         // DB_HOST
	port := viper.GetString("PORT")         // DB_PORT
	user := viper.GetString("USER")         // DB_USER
	password := viper.GetString("PASSWORD") // DB_PASSWORD
	dbname := viper.GetString("NAME")       // DB_NAME

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
