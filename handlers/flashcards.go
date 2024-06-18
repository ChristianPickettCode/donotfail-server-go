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
)

// GenerateFlashCards is a gin handler to generate flashcards from slide images
func GenerateFlashCards(c *gin.Context) {
	slideID := c.Param("slide_id")
	log.Println("*** /generate-flashcards ***")

	// Retrieve all slide images for the specified slide ID
	slideImages, err := findSlideImagesBySlideID(slideID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Found slide images:", len(slideImages))

	var allFlashcards []models.Flashcard
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
		// Every 10 slide images, generate flashcards
		if (i+1)%10 == 0 || i+1 == len(slideImages) {
			slideImageID := slideImage["_id"].(primitive.ObjectID).Hex()
			flashcards, err := generateFlashcards(contextStr, slideID, slideImageID)
			if err != nil {
				log.Println("Error generating flashcards", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			log.Println("Generated flashcards:", len(flashcards))

			// Store generated flashcards in the database
			if err := storeFlashcards(flashcards); err != nil {
				log.Println("Error storing flashcards", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			log.Println("Stored flashcards in the database")

			allFlashcards = append(allFlashcards, flashcards...)
			contextStr = "" // Reset context for next chunk

		}

	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": allFlashcards})
}

// Route to return list of slides based on if they have flashcards
func GetSlidesWithFlashcards(c *gin.Context) {
	log.Println("*** /slides-with-flashcards ***")
	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	// Find all slide IDs with flashcards
	slideIDs, err := db.DB.Collection("flashcards").Distinct(ctx, "slide_id", bson.M{})
	if err != nil {
		log.Println("Error finding slides with flashcards", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Found slides with flashcards")

	var slideIDStrings []string
	for _, id := range slideIDs {
		slideIDStrings = append(slideIDStrings, id.(string))
	}

	log.Println("Slide IDs with flashcards:", slideIDStrings)

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

// Delete Flashcard by Flashcard ID
func DeleteFlashcard(c *gin.Context) {
	flashcardID := c.Param("flashcard_id")
	log.Println("*** /flashcards ***")

	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	flashcardObjID, err := primitive.ObjectIDFromHex(flashcardID)
	if err != nil {
		log.Println("Error converting flashcard ID to ObjectID", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = db.DB.Collection("flashcards").DeleteOne(ctx, bson.M{"_id": flashcardObjID})
	if err != nil {
		log.Println("Error deleting flashcard", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// Get all flashcards for a slide
func GetFlashcards(c *gin.Context) {
	slideID := c.Param("slide_id")
	log.Println("*** /flashcards ***")

	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection("flashcards").Find(ctx, bson.M{"slide_id": slideID})
	if err != nil {
		log.Println("Error finding flashcards", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var flashcards []models.Flashcard
	if err = cursor.All(ctx, &flashcards); err != nil {
		log.Println("Error decoding flashcards", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": flashcards})
}

// Get all flashcards for a slide image
func GetFlashcardsForSlideImage(c *gin.Context) {
	slideID := c.Param("slide_id")
	slideImageID := c.Param("slide_image_id")
	log.Println("*** /flashcards ***")

	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection("flashcards").Find(ctx, bson.M{"slide_id": slideID, "slide_image_id": slideImageID})
	if err != nil {
		log.Println("Error finding flashcards", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var flashcards []models.Flashcard
	if err = cursor.All(ctx, &flashcards); err != nil {
		log.Println("Error decoding flashcards", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": flashcards})
}

func generateFlashcards(contextStr string, slideID string, slideImageID string) ([]models.Flashcard, error) {
	PROMPT := fmt.Sprintf(`
	You are a professor. Generate flashcards for university students to review the main concepts from the following content. Ensure the flashcards are relevant and based on the important topics of the slides, excluding any course administration or professor-related details. Assume the student does not have access to the slides when reviewing the flashcards. Each flashcard should have a question on one side and the corresponding answer on the other. Provide a rationale for the answer. Return the response in JUST JSON format array, nothing else.
	example: 
	"flashcards": [
        {
            "question": "What is the impact of France's urban planning policies on carbon emissions?",
            "answer": "France's urban planning has significantly reduced emissions by promoting public transportation and reducing car usage.",
            "rationale": "France's focus on urban planning, particularly in promoting public transportation, has led to a measurable decrease in car usage and emissions."
        }
    ]
	Content:
	%s
	`, contextStr)

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
	})

	if err != nil {
		return nil, err
	}

	log.Println("Response:", result.Choices[0].Message.Content)

	var flashcards []models.Flashcard
	for _, choice := range result.Choices {
		fcs, err := parseFlashcards(choice.Message.Content, slideID, slideImageID)
		if err != nil {
			return nil, err
		}
		flashcards = append(flashcards, fcs...)
	}

	return flashcards, nil
}

func parseFlashcards(content, slideID, slideImageID string) ([]models.Flashcard, error) {
	log.Println("Parsing flashcards")

	type Flashcards struct {
		Flashcards []models.Flashcard `json:"flashcards"`
	}

	var fc Flashcards

	err := json.Unmarshal([]byte(content), &fc)
	if err != nil {
		return nil, err
	}

	// Populate SlideID and SlideImageID in each flashcard
	for i := range fc.Flashcards {
		fc.Flashcards[i].ID = primitive.NewObjectID()
		fc.Flashcards[i].SlideID = slideID
		fc.Flashcards[i].SlideImageID = slideImageID
	}

	return fc.Flashcards, nil
}

func storeFlashcards(flashcards []models.Flashcard) error {
	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	var docs []interface{}
	for _, flashcard := range flashcards {
		docs = append(docs, flashcard)
	}

	_, err := db.DB.Collection("flashcards").InsertMany(ctx, docs)
	return err
}

func getFlashcardsForSlideImage(slideID string, slideImageID string) ([]models.Flashcard, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection("flashcards").Find(ctx, bson.M{"slide_id": slideID, "slide_image_id": slideImageID})
	if err != nil {
		return nil, fmt.Errorf("error finding flashcards: %v", err)
	}
	defer cursor.Close(ctx)

	var flashcards []models.Flashcard
	if err = cursor.All(ctx, &flashcards); err != nil {
		return nil, fmt.Errorf("error decoding flashcards: %v", err)
	}
	return flashcards, nil
}

// Generate flashcards for a specific slide image
func GenerateFlashcardsForSlideImage(c *gin.Context) {
	slideID := c.Param("slide_id")
	slideImageID := c.Param("slide_image_id")
	log.Println("*** /generate-flashcards ***")

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

	// Retrieve existing flashcards for the slide image
	existingFlashcards, err := getFlashcardsForSlideImage(slideID, slideImageID)
	if err != nil {
		log.Println("Error retrieving existing flashcards", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Prepare the context string with existing flashcards
	contextStr := generatedText
	if len(existingFlashcards) > 0 {
		contextStr += "\n\nExisting Flashcards:\n"
		for _, fc := range existingFlashcards {
			contextStr += fmt.Sprintf("Q: %s\nA: %s\n", fc.Question, fc.Answer)
		}
	}

	// Generate flashcards
	flashcards, err := generateFlashcards(contextStr, slideID, slideImageID)
	if err != nil {
		log.Println("Error generating flashcards", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Generated flashcards:", len(flashcards))

	// Store generated flashcards in the database
	if err := storeFlashcards(flashcards); err != nil {
		log.Println("Error storing flashcards", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Stored flashcards in the database")

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": flashcards})
}
