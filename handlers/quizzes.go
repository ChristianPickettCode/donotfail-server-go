package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"main/db"
	"main/models"
	"main/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GenerateQuizQuestions is a gin handler to generate quiz questions from slide images
func GenerateQuizQuestions(c *gin.Context) {
	slideID := c.Param("slide_id")
	log.Println("*** /generate-quiz-questions ***")

	// Retrieve all slide images for the specified slide ID
	slideImages, err := findSlideImagesBySlideIDQuiz(slideID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Found slide images:", len(slideImages))

	var allQuestions []models.QuizQA
	var contextStr string

	// Process in chunks of 10 slide images
	for i, slideImage := range slideImages {
		log.Println("Processing slide image", i+1)
		generatedText, ok := slideImage["generated_text"].(string)
		if !ok || generatedText == "" {
			log.Println("Generated text not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Generated text not found"})
			return
		}
		contextStr += fmt.Sprintf("Slide ID: %s, Slide Image ID: %s\n%s\n\n", slideID, slideImage["_id"].(primitive.ObjectID).Hex(), generatedText)

		log.Println("Context length:", len(contextStr))
		// Every 10 slide images, generate 20 quiz questions
		if (i+1)%5 == 0 || i+1 == len(slideImages) {
			slideImageID := slideImage["_id"].(primitive.ObjectID).Hex()
			questions, err := generateQuizQuestions(contextStr, slideID, slideImageID, 10)
			if err != nil {
				log.Println("Error generating quiz questions", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			log.Println("Generated questions:", len(questions))

			// Store generated questions in the database
			if err := storeQuizQuestions(questions); err != nil {
				log.Println("Error storing quiz questions", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			log.Println("Stored questions in the database")

			allQuestions = append(allQuestions, questions...)
			contextStr = "" // Reset context for next chunk

		}

		// if (i+1)%5 == 0 || i+1 == len(slideImages) {
		// 	break // For testing purposes, only process the first slide image
		// }

	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": allQuestions})
}

// Route to return list of slides based on if they have quiz questions
func GetSlidesWithQuizQuestions(c *gin.Context) {
	log.Println("*** /slides-with-quiz-questions ***")
	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	// Find all slide IDs with quiz questions
	slideIDs, err := db.DB.Collection("quiz_questions").Distinct(ctx, "slide_id", bson.M{})
	if err != nil {
		log.Println("Error finding slides with quiz questions", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Found slides with quiz questions")

	var slideIDStrings []string
	for _, id := range slideIDs {
		slideIDStrings = append(slideIDStrings, id.(string))
	}

	log.Println("Slide IDs with quiz questions:", slideIDStrings)

	// Get slide details
	var slides []models.Slide
	for _, slideID := range slideIDStrings {
		slide, err := findSlideByID(slideID)
		if err != nil {
			log.Println("Error finding slide by ID", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		slides = append(slides, slide)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": slides})
}

// Delete Quiz Question by Quiz ID
func DeleteQuizQuestion(c *gin.Context) {
	quizID := c.Param("quiz_id")
	log.Println("*** /quiz-questions ***")

	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	quizObjID, err := primitive.ObjectIDFromHex(quizID)
	if err != nil {
		log.Println("Error converting quiz ID to ObjectID", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = db.DB.Collection("quiz_questions").DeleteOne(ctx, bson.M{"_id": quizObjID})
	if err != nil {
		log.Println("Error deleting quiz questions", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// Get all quiz questions for a slide
func GetQuizQuestions(c *gin.Context) {
	slideID := c.Param("slide_id")
	log.Println("*** /quiz-questions ***")

	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection("quiz_questions").Find(ctx, bson.M{"slide_id": slideID})
	if err != nil {
		log.Println("Error finding quiz questions", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var questions []models.QuizQA
	if err = cursor.All(ctx, &questions); err != nil {
		log.Println("Error decoding quiz questions", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": questions})
}

// Get all quiz questions for a slide image
func GetQuizQuestionsForSlideImage(c *gin.Context) {
	slideID := c.Param("slide_id")
	slideImageID := c.Param("slide_image_id")
	log.Println("*** /quiz-questions ***")

	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection("quiz_questions").Find(ctx, bson.M{"slide_id": slideID, "slide_image_id": slideImageID})
	if err != nil {
		log.Println("Error finding quiz questions", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var questions []models.QuizQA
	if err = cursor.All(ctx, &questions); err != nil {
		log.Println("Error decoding quiz questions", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": questions})
}

func findSlideByID(slideID string) (models.Slide, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(slideID)
	if err != nil {
		return models.Slide{}, err
	}

	var slide models.Slide
	err = db.DB.Collection("slides").FindOne(ctx, bson.M{"_id": objID}).Decode(&slide)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.Slide{}, fmt.Errorf("slide not found")
		}
		return models.Slide{}, fmt.Errorf("error finding slide: %v", err)
	}
	return slide, nil
}

func findSlideImagesBySlideIDQuiz(slideID string) ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	opts := options.Find()
	cursor, err := db.DB.Collection("slide_images").Find(ctx, bson.M{"slide_id": slideID}, opts)
	if err != nil {
		log.Println("Error finding slide images", err)
		return nil, fmt.Errorf("error finding slide images: %v", err)
	}
	defer cursor.Close(ctx)

	var slideImages []bson.M
	if err = cursor.All(ctx, &slideImages); err != nil {
		log.Println("Error decoding slide images", err)
		return nil, fmt.Errorf("error decoding slide images: %v", err)
	}
	return slideImages, nil
}

func generateQuizQuestions(contextStr string, slideID string, slideImageID string, numOfQ int) ([]models.QuizQA, error) {
	strNumQ := fmt.Sprintf("%d", numOfQ)
	PROMPT := fmt.Sprintf(`
	You are a professor. You MUST using Bloom's revised taxonomy, generate %s quiz questions at level 5 (evaluate) for a university student who wants to review the main concepts of the learning objectives from the following content. Ensure the questions are relevant and based on the important topics of the slides, excluding any course administration or professor-related questions. Assume the student does not have access to the slides when completing the quiz. Each question should have 4 answer choices and specify the correct answer. Return the response in JUST JSON format array, nothing else. If there are existing questions, generate questions for other parts of the content. Make sure the answers are clear, 3-4 sentences long, and provide a rationale for the correct answer. Do not include any questions that are too similar to existing questions. Bloom's revised taxonomy, level 5 (evaluate) requires students to make judgments based on criteria and standards.
	example: 
	"quiz_questions": [
        {
            "question": "Assess the effectiveness of France's approach to urban planning in reducing carbon emissions compared to Germany's strategies.",
            "answer_choices": [
                "France's approach is more effective due to its focus on public transportation.",
                "Germany's approach is more effective due to its emphasis on renewable energy.",
                "Both approaches are equally effective but in different areas.",
                "Neither approach has been effective in reducing carbon emissions."
            ],
            "answer": "France's approach is more effective due to its focus on public transportation.",
            "slide_id": "slide_id_here",
            "slide_image_id": "slide_image_id_here"
        }
    ]
	Content:
	%s
	`, strNumQ, contextStr)

	log.Println("Prompt:", PROMPT)

	client := openai.NewClient(utils.OPENAI_API_KEY)
	result, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: PROMPT,
			},
		},
		MaxTokens: 4000,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})

	if err != nil {
		return nil, err
	}

	log.Println("Response:", result.Choices[0].Message.Content)

	var questions []models.QuizQA
	for _, choice := range result.Choices {
		qas, err := parseQuizQuestions(choice.Message.Content, slideID, slideImageID)
		if err != nil {
			return nil, err
		}
		questions = append(questions, qas...)
	}

	return questions, nil
}

func parseQuizQuestions(content, slideID, slideImageID string) ([]models.QuizQA, error) {
	log.Println("Parsing quiz questions")
	// var questions []models.QuizQA

	// quiz_questions struct
	type QuizQuestions struct {
		QuizQuestions []models.QuizQA `json:"quiz_questions"`
	}

	var qq QuizQuestions

	// Clean the JSON string
	// cleanedContent := cleanJSONString(content)

	// parse the JSON string quiz_questions

	// Assuming the content is a JSON string, unmarshal it
	err := json.Unmarshal([]byte(content), &qq)
	if err != nil {
		return nil, err
	}

	// Populate SlideID and SlideImageID in each question
	for i := range qq.QuizQuestions {
		qq.QuizQuestions[i].ID = primitive.NewObjectID()
		qq.QuizQuestions[i].SlideID = slideID
		qq.QuizQuestions[i].SlideImageID = slideImageID // Set the SlideImageID
	}

	return qq.QuizQuestions, nil
}

func storeQuizQuestions(questions []models.QuizQA) error {
	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	var docs []interface{}
	for _, question := range questions {
		docs = append(docs, question)
	}

	_, err := db.DB.Collection("quiz_questions").InsertMany(ctx, docs)
	return err
}

func getQuizQuestionsForSlideImage(slideID string, slideImageID string) ([]models.QuizQA, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection("quiz_questions").Find(ctx, bson.M{"slide_id": slideID, "slide_image_id": slideImageID})
	if err != nil {
		return nil, fmt.Errorf("error finding quiz questions: %v", err)
	}
	defer cursor.Close(ctx)

	var questions []models.QuizQA
	if err = cursor.All(ctx, &questions); err != nil {
		return nil, fmt.Errorf("error decoding quiz questions: %v", err)
	}
	return questions, nil
}

// Generate quiz questions for a specific slide image
func GenerateQuizQuestionsForSlideImage(c *gin.Context) {
	slideID := c.Param("slide_id")
	slideImageID := c.Param("slide_image_id")
	log.Println("*** /generate-quiz-questions ***")

	objID, err := primitive.ObjectIDFromHex(slideImageID)
	if err != nil {
		log.Println("Error converting slide image ID to ObjectID", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Retrieve slide image
	slideImage, err := findSlideImageByID(objID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Found slide image")

	generatedText, ok := slideImage["generated_text"].(string)
	if !ok || generatedText == "" {
		log.Println("Generated text not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Generated text not found"})
		return
	}

	log.Println("Generated text:", generatedText)

	// Retrieve existing questions for the slide image
	existingQuestions, err := getQuizQuestionsForSlideImage(slideID, slideImageID)
	if err != nil {
		log.Println("Error retrieving existing quiz questions", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Prepare the context string with existing questions
	contextStr := generatedText
	if len(existingQuestions) > 0 {
		contextStr += "\n\nExisting Questions:\n"
		for _, q := range existingQuestions {
			contextStr += fmt.Sprintf("Q: %s\nA: %s\n", q.Question, q.Answer)
		}
	}

	// Generate quiz questions
	questions, err := generateQuizQuestions(contextStr, slideID, slideImageID, 3)
	if err != nil {
		log.Println("Error generating quiz questions", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Generated questions:", len(questions))

	// Store generated questions in the database
	if err := storeQuizQuestions(questions); err != nil {
		log.Println("Error storing quiz questions", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Stored questions in the database")

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": questions})
}
