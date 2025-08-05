package models

import (
	"encoding/json"
	"time"
)

type User struct {
	ID         uint   `gorm:"primaryKey"`
	GoogleID   string `gorm:"unique;not null"`
	Email      string `gorm:"unique;not null"`
	Name       string `gorm:"not null"`
	PictureURL string
	CreatedAt  time.Time
}

type UserTest struct {
	TestID            uint            `gorm:"primaryKey"`
	UserID            uint            `gorm:"not null"`
	AnswerKey         json.RawMessage `gorm:"type:jsonb"`
	CorrectionResults json.RawMessage `gorm:"type:jsonb"`
	CreatedAt         time.Time

	User    User    `gorm:"foreignKey:UserID"`
	AiScore AiScore `gorm:"foreignKey:TestID"`
}

type AiScore struct {
	ID                    uint            `gorm:"primaryKey"`
	TestID                uint            `gorm:"unique;not null"`
	Diagnosis             int             `gorm:"not null"`
	FinalDyscalculiaScore float64         `gorm:"not null"`
	RawResponse           json.RawMessage `gorm:"type:jsonb"`
	CreatedAt             time.Time
}

type Question struct {
	ID           uint            `gorm:"primaryKey"`
	SubtestName  string          `gorm:"not null"`
	QuestionID   string          `gorm:"unique;not null"`
	QuestionData json.RawMessage `gorm:"type:jsonb;not null"`
	AnswerData   string          `gorm:"not null"`
	CreatedAt    time.Time
}
