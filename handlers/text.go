package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"main/db"
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

func GenerateText(c *gin.Context) {
	slideImageID := c.Param("slide_image_id")
	fmt.Println("*** /generate-image-text ***")

	objID, err := primitive.ObjectIDFromHex(slideImageID)
	if err != nil {
		log.Println("Invalid slide image ID", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slide image ID"})
		return
	}

	slideImage, err := findSlideImageByID(objID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	imageURL, ok := slideImage["image_url"].(string)
	if !ok || imageURL == "" {
		log.Println("Image not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	contextStr, err := generateContextForSlideImage(slideImage)
	if err != nil {
		log.Println("Error generating context", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	sendSSE := func(message string) {
		fmt.Fprintf(c.Writer, "data: %s\n\n", message)
		c.Writer.Flush()
	}

	sendSSE("Processing image to generate text")

	response, err := processImage(imageURL, contextStr)
	if err != nil {
		log.Println("Error processing image", err)
		sendSSE("Error processing image")
		return
	}

	sendSSE("Updating generated text in the database")

	if !updateGeneratedText(slideImageID, response) {
		log.Println("Error updating slide image")
		sendSSE("Error updating slide image")
		return
	}

	finalResponse, _ := json.Marshal(gin.H{"status": "success", "data": response})
	fmt.Fprintf(c.Writer, "data: %s\n\n", finalResponse)
	c.Writer.Flush()
}

func GenerateAllImageText(c *gin.Context) {
	slideID := c.Param("slide_id")
	log.Println("*** /generate-all-image-text ***")

	log.Println("Slide ID:", slideID)

	slideImages, err := findSlideImagesBySlideID(slideID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	sendSSE := func(message string) {
		fmt.Fprintf(c.Writer, "data: %s\n\n", message)
		c.Writer.Flush()
	}

	totalImages := len(slideImages)
	sendSSE(fmt.Sprintf(`{"totalImages": %d}`, totalImages))

	var slideImagesList []bson.M
	for _, slideImage := range slideImages {
		imageURL, ok := slideImage["image_url"].(string)
		if !ok || imageURL == "" {
			log.Println("Image URL not found")
			sendSSE("Image URL not found")
			continue
		}

		generatedText, _ := slideImage["generated_text"].(string)
		if generatedText == "" {
			contextStr, err := generateContextForSlideImage(slideImage)
			if err != nil {
				log.Println("Error generating context", err)
				sendSSE("Error generating context")
				continue
			}

			sendSSE(fmt.Sprintf("Processing image for slide order %d", slideImage["order"].(int32)))

			response, err := processImage(imageURL, contextStr)
			if err != nil {
				log.Println("Error processing image", err)
				sendSSE(fmt.Sprintf("Error processing image for slide order %d", slideImage["order"].(int32)))
				continue
			}

			if !updateGeneratedText(slideImage["_id"].(primitive.ObjectID).Hex(), response) {
				log.Println("Error updating slide image")
				sendSSE(fmt.Sprintf("Error updating slide image for slide order %d", slideImage["order"].(int32)))
				continue
			}
			slideImage["generated_text"] = response
		}

		slideImagesList = append(slideImagesList, slideImage)
		sendSSE(fmt.Sprintf(`{"processedImage": %d}`, slideImage["order"].(int32)))
	}

	finalResponse, _ := json.Marshal(gin.H{"status": "success", "data": slideImagesList})
	fmt.Fprintf(c.Writer, "data: %s\n\n", finalResponse)
	sendSSE("[DONE]")
	c.Writer.Flush()
}

// findSlideImageByID retrieves a single slide image by ID from the database
func findSlideImageByID(id primitive.ObjectID) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var slideImage bson.M
	slideImageResult := db.DB.Collection(collectionNameSlideImages).FindOne(ctx, bson.M{"_id": id})
	if err := slideImageResult.Decode(&slideImage); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("slide Image not found")
		}
		return nil, fmt.Errorf("error finding slide image: %v", err)
	}
	return slideImage, nil
}

// findSlideImagesBySlideID retrieves all slide images for a given slide ID from the database
func findSlideImagesBySlideID(slideID string) ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	opts := options.Find()
	opts.SetSort(bson.D{{Key: "order", Value: 1}}) // Sort by 'order' field in ascending order

	cursor, err := db.DB.Collection(collectionNameSlideImages).Find(ctx, bson.M{"slide_id": slideID}, opts)
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

// generateContextForSlideImage generates the context string for a slide image based on preceding slides
func generateContextForSlideImage(slideImage bson.M) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()

	slideID := slideImage["slide_id"].(string)
	slideOrder := slideImage["order"].(int32)

	// Find preceding 2 slides
	precedingSlidesCursor, err := db.DB.Collection(collectionNameSlideImages).Find(ctx, bson.M{"slide_id": slideID, "order": bson.M{"$lt": slideOrder}})
	if err != nil {
		return "", fmt.Errorf("error finding preceding slides: %v", err)
	}
	defer precedingSlidesCursor.Close(ctx)

	var precedingSlides []bson.M
	for precedingSlidesCursor.Next(ctx) {
		var slide bson.M
		if err := precedingSlidesCursor.Decode(&slide); err != nil {
			return "", fmt.Errorf("error decoding preceding slide: %v", err)
		}
		precedingSlides = append(precedingSlides, slide)
	}

	var contextStr string
	if len(precedingSlides) >= 2 {
		contextStr += fmt.Sprintf("SLIDE %d: \n%s\n\n", precedingSlides[len(precedingSlides)-2]["order"].(int32)+1, precedingSlides[len(precedingSlides)-2]["generated_text"].(string))
		contextStr += fmt.Sprintf("SLIDE %d: \n%s\n\n", precedingSlides[len(precedingSlides)-1]["order"].(int32)+1, precedingSlides[len(precedingSlides)-1]["generated_text"].(string))
	} else if len(precedingSlides) == 1 {
		contextStr += fmt.Sprintf("SLIDE %d: \n%s\n\n", precedingSlides[0]["order"].(int32)+1, precedingSlides[0]["generated_text"].(string))
	}

	contextStr += fmt.Sprintf("SLIDE %d: \n", slideOrder+1)
	return contextStr, nil
}

// processImage calls the API to process the image and generate text
func processImage(imageURL string, contextStr string) (string, error) {
	PROMPT := `
	
	You are a professor, describe and explain this lecture slide, no fluff, buzzwords or jargon. Use the context(previous slides) provided to give a clear and concise explanation of this current slide.
	Do not start explanation with 'this slide', or 'the slide', or 'the title', or 'the presentation', 'Today's lecture' or statements like those, just start explaining the slide.
	Don't make up information, only use the information provided in the slide and expand if necessary for clarity and understanding. Make the transitions between slides smooth and coherent as if you were giving a lecture. Do not use the words 'delve', or 'slide'. Start the explanation as if you were continuing from the previous slide.  Bold the keywords and key phrases in your explanation. 
	`
	PROMPT = contextStr + PROMPT
	fmt.Println("PROMPT: ", PROMPT)
	return callAPI(imageURL, PROMPT)
}

// callAPI makes the API call to generate text based on the image and prompt
func callAPI(imageURL string, prompt string) (string, error) {
	client := openai.NewClient(utils.OPENAI_API_KEY)

	result, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: prompt,
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL:    imageURL,
							Detail: openai.ImageURLDetailHigh,
						},
					},
				},
			},
		},
		MaxTokens: 3000,
	})

	if err != nil {
		return "", err
	}

	for _, choice := range result.Choices {
		log.Println("Choice:", choice.Message.Content)

		for _, part := range choice.Message.MultiContent {
			if part.Type == openai.ChatMessagePartTypeText {
				log.Println("Partial text:", part.Text)
			}
		}
	}

	return result.Choices[0].Message.Content, nil
}

// updateGeneratedText updates the generated text in the database for a given slide image ID
func updateGeneratedText(slideImageID string, generatedText string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()
	objID, err := primitive.ObjectIDFromHex(slideImageID)
	if err != nil {
		log.Println("Invalid slide image ID", err)
		return false
	}
	_, err = db.DB.Collection(collectionNameSlideImages).UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": bson.M{"generated_text": generatedText}})
	if err != nil {
		log.Println("Error updating generated text", err)
		return false
	}
	return true
}

func GenerateNotes(c *gin.Context) {
	slideID := c.Param("slide_id")
	log.Println("*** /generate-notes ***")

	slideImages, err := findSlideImagesBySlideID(slideID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var generatedNotes []string
	for _, slideImage := range slideImages {
		generatedText, ok := slideImage["generated_text"].(string)
		if !ok || generatedText == "" {
			log.Println("Generated text not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Generated text not found"})
			return
		}

		generatedNotes = append(generatedNotes, generatedText)
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": generatedNotes})
}
