package handlers

import (
	"Dysec/internal/ai"
	"Dysec/internal/models"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
	//"gorm.io/gorm/clause"
)

type SubtestPayload struct {
	Answers         map[string]string      `json:"answers"`
	PerformanceData map[string]interface{} `json:"performance_data"`
}
type SubmitRequest struct {
	SimpleReactionTime SubtestPayload `json:"simple_reaction_time"`
	Dot                SubtestPayload `json:"dot"`
	Stroop             SubtestPayload `json:"stroop"`
	Addition           SubtestPayload `json:"addition"`
	Multiplication     SubtestPayload `json:"multiplication"`
	Substitution       SubtestPayload `json:"substitution"`
}

type GoogleAuthRequest struct {
	Token      string `json:"token"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	GoogleID   string `json:"google_id"`
	PictureURL string `json:"picture_url"`
}

type AIResponse struct {
	Questions json.RawMessage `json:"questions"`
	AnswerKey json.RawMessage `json:"answer_key"`
}

type Handler struct {
	DB        *gorm.DB
	AIService *ai.Service
}

type AIRequestData struct {
	Age       int     `json:"age"`
	Srt       float64 `json:"srt"`
	DotRt     float64 `json:"dot_rt"`
	DotAcc    float64 `json:"dot_acc"`
	StroopRt  float64 `json:"stroop_rt"`
	StroopAcc float64 `json:"stroop_acc"`
	AddRt     float64 `json:"add_rt"`
	AddAcc    float64 `json:"add_acc"`
	MultRt    float64 `json:"mult_rt"`
	MultAcc   float64 `json:"mult_acc"`
	SubsRt    float64 `json:"subs_rt"`
	SubsAcc   float64 `json:"subs_acc"`
}

func New(db *gorm.DB, aiService *ai.Service) Handler {
	return Handler{DB: db, AIService: aiService}
}

func (h *Handler) GoogleAuthHandler(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: token is required"})
		return
	}

	// Verifikasi token untuk memastikan login valid dan mendapatkan data user
	payload, err := idtoken.Validate(c.Request.Context(), req.Token, "")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Google ID Token"})
		return
	}

	// Ambil data dari token dan simpan/update user di database
	user := models.User{
		GoogleID:   payload.Subject,
		Email:      payload.Claims["email"].(string),
		Name:       payload.Claims["name"].(string),
		PictureURL: payload.Claims["picture"].(string),
	}

	if err := h.DB.Where(models.User{GoogleID: user.GoogleID}).FirstOrCreate(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not process user"})
		return
	}

	// Tidak ada lagi app_token yang dikembalikan
	c.JSON(http.StatusOK, gin.H{
		"message": "User authenticated successfully. Use the provided ID token for subsequent requests.",
		"user":    user,
	})
}

/*
// GoogleAuthHandler versi produksi dengan verifikasi token Google

	func (h *Handler) GoogleAuthHandler(c *gin.Context) {
		var req struct {
			Token string `json:"token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: token is required"})
			return
		}

		payload, err := idtoken.Validate(c.Request.Context(), req.Token, "")
		if err != nil {
			log.Printf("ERROR: Invalid Google ID Token: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Google ID Token"})
			return
		}

		googleID := payload.Subject
		email := payload.Claims["email"].(string)
		name := payload.Claims["name"].(string)
		pictureURL := payload.Claims["picture"].(string)

		user := models.User{
			GoogleID:   googleID,
			Email:      email,
			Name:       name,
			PictureURL: pictureURL,
		}

		if err := h.DB.Where(models.User{GoogleID: googleID}).FirstOrCreate(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not process user"})
			return
		}

		claims := jwt.MapClaims{
			"user_id": user.ID,
			"email":   user.Email,
			"exp":     time.Now().Add(time.Hour * 72).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(h.JWTSecret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "User authenticated successfully",
			"app_token": tokenString,
		})
	}
*/
func (h *Handler) StartSessionHandler(c *gin.Context) {
	userIDClaim, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}
	userID := uint(userIDClaim.(float64))

	var finalSubtestsData json.RawMessage
	var finalError error

	fallbackQuestionCount := map[string]int{
		"dot":            2,
		"stroop":         2,
		"addition":       2,
		"multiplication": 2,
		"substitution":   2,
	}

	prompt := `Buatkan satu set soal tes untuk deteksi gejala diskalkulia yang terdiri dari 6 subtes: simple_reaction_time, dot, stroop, addition, multiplication, dan substitution.

Output HARUS dalam format JSON tunggal dengan satu key utama "subtests".
Setiap objek subtes HARUS memiliki dua key: "questions" dan "answer_key".
Setiap soal di dalam array "questions" HARUS memiliki key "question_id" dengan nilai string yang unik.

- Untuk subtes tanpa pertanyaan (simple_reaction_time), "questions" harus berupa array kosong [].
- Untuk subtes dengan pertanyaan, "questions" harus berupa array of objects, dan "answer_key" harus berupa object map dimana key-nya adalah "question_id" yang sesuai.

Buatkan 2 soal untuk setiap subtes "dot", "stroop", "addition", "multiplication", dan "substitution".

Contoh Format yang WAJIB diikuti:
{
  "subtests": {
    "addition": {
        "questions": [
            {"question_id": "add_1", "type": "text_input", "text": "Berapa 12 + 9?"},
            {"question_id": "add_2", "type": "text_input", "text": "Berapa 7 + 6?"}
        ],
        "answer_key": {
            "add_1": "21",
            "add_2": "13"
        }
    }
  }
}`
	// Alur 1: Coba panggil AI Service
	aiResponseJSON, err := h.AIService.GenerateTestFromPrompt(prompt)
	if err == nil {
		log.Println("INFO: AI call successful. Starting to process response...")

		startIndex := strings.Index(aiResponseJSON, "{")
		endIndex := strings.LastIndex(aiResponseJSON, "}")

		if startIndex != -1 && endIndex != -1 && endIndex > startIndex {
			cleanJSON := aiResponseJSON[startIndex : endIndex+1]
			log.Println("INFO: Successfully cleaned JSON response.")

			var parsedAIResponse map[string]json.RawMessage
			if err := json.Unmarshal([]byte(cleanJSON), &parsedAIResponse); err == nil {
				log.Println("INFO: Successfully unmarshaled root JSON.")

				if subtestsJSON, ok := parsedAIResponse["subtests"]; ok {
					log.Println("INFO: 'subtests' key found. Proceeding to save questions.")
					finalSubtestsData = subtestsJSON

					var subtests map[string]struct {
						Questions []map[string]interface{} `json:"questions"`
						AnswerKey map[string]interface{}   `json:"answer_key"`
					}
					if err := json.Unmarshal(subtestsJSON, &subtests); err == nil {
						log.Printf("INFO: Parsed %d subtests. Looping to save individual questions...", len(subtests))
						for subtestName, data := range subtests {
							log.Printf("INFO: Processing subtest: %s", subtestName)
							for _, q := range data.Questions {
								qID, id_ok := q["question_id"].(string)
								if !id_ok {
									log.Println("WARNING: question_id not found or not a string, skipping question.")
									continue
								}

								var existingQuestion models.Question
								dbErr := h.DB.Where("question_id = ?", qID).First(&existingQuestion).Error

								if errors.Is(dbErr, gorm.ErrRecordNotFound) {
									answer := fmt.Sprintf("%v", data.AnswerKey[qID])
									qData, _ := json.Marshal(q)

									newQuestion := models.Question{
										SubtestName:  subtestName,
										QuestionID:   qID,
										QuestionData: qData,
										AnswerData:   answer,
									}
									if createErr := h.DB.Create(&newQuestion).Error; createErr != nil {
										log.Printf("ERROR: Failed to create new question %s: %v", qID, createErr)
									} else {
										log.Printf("INFO: Successfully saved new question %s to database.", qID)
									}
								} else if dbErr == nil {
									log.Printf("INFO: Question %s already exists in DB, skipping.", qID)
								} else {
									log.Printf("ERROR: DB check failed for question %s: %v", qID, dbErr)
								}
							}
						}
					} else {
						log.Printf("ERROR: Failed to unmarshal the 'subtests' block: %v", err)
						finalError = fmt.Errorf("failed to unmarshal subtests block: %w", err)
					}
				} else {
					log.Println("ERROR: 'subtests' key was not found in the parsed JSON.")
					finalError = fmt.Errorf("key 'subtests' not found in AI response")
				}
			} else {
				log.Printf("ERROR: Failed to unmarshal the cleaned root JSON: %v", err)
				finalError = fmt.Errorf("failed to parse AI response JSON: %w", err)
			}
		} else {
			log.Println("ERROR: Could not find valid JSON object in AI response string.")
			finalError = fmt.Errorf("could not find valid JSON object in AI response")
		}
	} else {
		finalError = err
	}

	// Alur 2: Jika ada error dari AI ATAU dari parsing, bangun tes fallback
	if finalError != nil {
		log.Printf("WARNING: An error occurred (%v). Building fallback test from database.", finalError)

		fallbackSubtests := make(map[string]interface{})
		for subtestName, count := range fallbackQuestionCount {
			var randomQuestions []models.Question
			h.DB.Where("subtest_name = ?", subtestName).Order("RANDOM()").Limit(count).Find(&randomQuestions)

			questionsForJSON := []json.RawMessage{}
			answersForJSON := make(map[string]string)
			for _, q := range randomQuestions {
				questionsForJSON = append(questionsForJSON, q.QuestionData)
				answersForJSON[q.QuestionID] = q.AnswerData
			}
			fallbackSubtests[subtestName] = map[string]interface{}{
				"questions":  questionsForJSON,
				"answer_key": answersForJSON,
			}
		}
		fallbackSubtests["simple_reaction_time"] = map[string]interface{}{
			"questions":  []interface{}{},
			"answer_key": map[string]string{},
		}

		finalSubtestsData, _ = json.Marshal(fallbackSubtests)
	}

	// Lanjutkan alur dengan data yang sudah didapat
	test := models.UserTest{
		UserID:    userID,
		AnswerKey: finalSubtestsData,
	}

	if err := h.DB.Create(&test).Error; err != nil {
		log.Printf("ERROR: Could not create test record: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start test"})
		return
	}

	finalResponse, _ := json.Marshal(map[string]interface{}{
		"message":  "Test started successfully",
		"test_id":  test.TestID,
		"subtests": finalSubtestsData,
	})

	c.Data(http.StatusOK, "application/json; charset=utf-8", finalResponse)
}
func (h *Handler) SubmitTestHandler(c *gin.Context) {

	testIDStr := c.Param("id")
	testID, _ := strconv.ParseUint(testIDStr, 10, 64)
	userIDClaim, _ := c.Get("user_id")
	userID := uint(userIDClaim.(float64))

	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body structure"})
		return
	}

	var test models.UserTest
	if err := h.DB.Where("test_id = ? AND user_id = ?", testID, userID).First(&test).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Test not found or you do not have permission"})
		return
	}

	var allSubtestsFromDB map[string]map[string]interface{}
	if err := json.Unmarshal(test.AnswerKey, &allSubtestsFromDB); err != nil {
		log.Printf("ERROR: Could not unmarshal answer key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not process answer key"})
		return
	}

	type CorrectionResult struct {
		Correct int `json:"correct"`
		Wrong   int `json:"wrong"`
		Total   int `json:"total"`
	}
	allCorrectionResults := make(map[string]CorrectionResult)

	corrector := func(subtestName string, userAnswers map[string]string) {
		if subtestData, ok := allSubtestsFromDB[subtestName]; ok {
			if answerKeyData, ok := subtestData["answer_key"].(map[string]interface{}); ok {
				correct, wrong := 0, 0
				for qID, correctAnswer := range answerKeyData {
					correctAnswerStr := fmt.Sprintf("%v", correctAnswer)
					if userAnswer, submitted := userAnswers[qID]; submitted && userAnswer == correctAnswerStr {
						correct++
					} else {
						wrong++
					}
				}
				allCorrectionResults[subtestName] = CorrectionResult{Correct: correct, Wrong: wrong, Total: len(answerKeyData)}
			}
		}
	}

	corrector("simple_reaction_time", req.SimpleReactionTime.Answers)
	corrector("dot", req.Dot.Answers)
	corrector("stroop", req.Stroop.Answers)
	corrector("addition", req.Addition.Answers)
	corrector("multiplication", req.Multiplication.Answers)
	corrector("substitution", req.Substitution.Answers)

	correctionJSON, _ := json.Marshal(allCorrectionResults)
	test.CorrectionResults = correctionJSON
	h.DB.Save(&test)

	var aiRequest AIRequestData

	// Fungsi bantu untuk konversi yang aman
	getFloat := func(data map[string]interface{}, key string) float64 {
		if val, ok := data[key].(float64); ok {
			return val
		}
		return 0.0
	}

	// Lakukan transformasi dengan aman
	aiRequest.Age = int(getFloat(req.SimpleReactionTime.PerformanceData, "age"))
	aiRequest.Srt = getFloat(req.SimpleReactionTime.PerformanceData, "median_reaction_time")
	aiRequest.DotRt = getFloat(req.Dot.PerformanceData, "median_reaction_time")
	if total := allCorrectionResults["dot"].Total; total > 0 {
		aiRequest.DotAcc = float64(allCorrectionResults["dot"].Correct) / float64(total)
	}
	aiRequest.StroopRt = getFloat(req.Stroop.PerformanceData, "median_reaction_time")
	if total := allCorrectionResults["stroop"].Total; total > 0 {
		aiRequest.StroopAcc = float64(allCorrectionResults["stroop"].Correct) / float64(total)
	}
	aiRequest.AddRt = getFloat(req.Addition.PerformanceData, "median_reaction_time")
	if total := allCorrectionResults["addition"].Total; total > 0 {
		aiRequest.AddAcc = float64(allCorrectionResults["addition"].Correct) / float64(total)
	}
	aiRequest.MultRt = getFloat(req.Multiplication.PerformanceData, "median_reaction_time")
	if total := allCorrectionResults["multiplication"].Total; total > 0 {
		aiRequest.MultAcc = float64(allCorrectionResults["multiplication"].Correct) / float64(total)
	}
	aiRequest.SubsRt = getFloat(req.Substitution.PerformanceData, "median_reaction_time")
	if total := allCorrectionResults["substitution"].Total; total > 0 {
		aiRequest.SubsAcc = float64(allCorrectionResults["substitution"].Correct) / float64(total)
	}

	log.Printf("INFO: Simulating call to AI for scoring with data: %+v\n", aiRequest)
	simulatedAIResponseStr := `{
		"diagnosis": 2, "label": {"diagnosis": {"0": "Normal", "1": "Diskalkulia", "2": "Keterampilan Aritmatika yang Buruk"}}, "probabilitas": {"0": 0.04, "1": 0.43, "2": 0.43}, "skor": [3.41, 3.3, 5.23, 1.58, 1.87]
	}`

	var aiResponseData map[string]interface{}
	json.Unmarshal([]byte(simulatedAIResponseStr), &aiResponseData)

	diagnosis := int(aiResponseData["diagnosis"].(float64))
	finalScore := aiResponseData["probabilitas"].(map[string]interface{})[fmt.Sprintf("%.0f", float64(diagnosis))].(float64)

	aiScore := models.AiScore{
		TestID:                test.TestID,
		Diagnosis:             diagnosis,
		FinalDyscalculiaScore: finalScore,
		RawResponse:           []byte(simulatedAIResponseStr),
	}
	h.DB.Create(&aiScore)

	// 9. Kirim hasil lengkap ke frontend
	c.JSON(http.StatusOK, gin.H{
		"message":            "Test submitted and graded successfully",
		"correction_results": allCorrectionResults,
		"ai_results":         aiResponseData,
	})
}
func (h *Handler) TestHistoryHandler(c *gin.Context) {
	userIDClaim, _ := c.Get("user_id")
	userID := uint(userIDClaim.(float64))

	var tests []models.UserTest
	err := h.DB.Preload("AiScore").Where("user_id = ?", userID).Order("created_at desc").Find(&tests).Error
	if err != nil {
		log.Printf("ERROR: Could not fetch test history: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch test history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": tests,
	})
}

func (h *Handler) UserProfileHandler(c *gin.Context) {
	userIDClaim, _ := c.Get("user_id")
	userID := uint(userIDClaim.(float64))

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          user.ID,
		"name":        user.Name,
		"email":       user.Email,
		"picture_url": user.PictureURL,
		"joined_at":   user.CreatedAt,
	})
}
