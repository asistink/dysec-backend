package ai

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/genai"
)

type Service struct {
	Client *genai.Client
}

func NewService(apiKey string) (*Service, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuat client genai: %w", err)
	}

	log.Println("AI Service Initialized Successfully")
	return &Service{Client: client}, nil
}

func (s *Service) GenerateTestFromPrompt(prompt string) (string, error) {
	ctx := context.Background()

	result, err := s.Client.Models.GenerateContent(
		ctx,
		"gemini-1.5-flash",
		genai.Text(prompt),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("gagal menghasilkan konten dari AI: %w", err)
	}

	text := result.Text()
	if text == "" {
		return "", fmt.Errorf("tidak ada konten teks yang ditemukan dalam respons AI")
	}

	return text, nil
}
