package handlers

// GenerateAudio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"main/db"
	"main/models"
	"main/utils"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var collectionNameSlideImages = "slide_images"

func GenerateAudio(c *gin.Context) {
	// Get request body
	var request models.AudioRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Println("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Print log
	log.Println("*** /generate-audio ***")

	// Get slide image ID
	slideImageID := request.SlideImageID
	log.Printf("Slide Image ID: %s", slideImageID)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(slideImageID)
	if err != nil {
		log.Println("Invalid slide image ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slide image ID"})
		return
	}

	// Find slide image
	var slideImage bson.M
	slideImageResult := db.DB.Collection(collectionNameSlideImages).FindOne(ctx, bson.M{"_id": objID})
	if err := slideImageResult.Decode(&slideImage); err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("Slide Image not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Slide Image not found"})
		} else {
			log.Println("Error finding slide image")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding slide image"})
		}
		return
	}

	// Get generated text
	generatedText, ok := slideImage["generated_text"].(string)
	if !ok {
		log.Println("Generated text not found")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Generated text not found"})
		return
	}

	// Check if audio URL already exists
	audioURL, ok := slideImage["audio_url"].(string)
	if ok && audioURL != "" && !request.Update {
		log.Println("Audio URL already exists")
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": audioURL, "status_code": http.StatusOK})
		return
	}

	// Generate audio file
	log.Println("Generating audio file")
	voice := "aura-athena-en"
	url := fmt.Sprintf("https://api.deepgram.com/v1/speak?model=%s", voice)
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Token %s", utils.DEEPGRAM_API_KEY),
	}
	data := map[string]string{
		"text": generatedText,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println("Error preparing request data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error preparing request data"})
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error creating request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating request"})
		return
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		log.Println("Error generating audio file")
		c.JSON(response.StatusCode, gin.H{"error": string(body)})
		return
	}

	log.Println("Audio file generated")

	// Read audio content
	audioBlob, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error reading audio response")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading audio response"})
		return
	}
	log.Println("Audio generated")

	// Upload audio to S3
	fileName := generateFileName()
	awsPath := fmt.Sprintf("slides/%s/audio/%s.mp3", slideImage["slide_id"].(string), fileName)
	s3session := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(utils.AWS_REGION),
	}))

	uploader := s3.New(s3session)
	_, err = uploader.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(utils.AWS_BUCKET_NAME),
		Key:         aws.String(awsPath),
		Body:        bytes.NewReader(audioBlob),
		ContentType: aws.String("audio/mpeg"),
	})
	if err != nil {
		log.Println("Error uploading audio to S3")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error uploading audio to S3"})
		return
	}

	log.Println("Audio uploaded to S3")

	// Generate audio URL
	audioURL = fmt.Sprintf("https://%s.s3.amazonaws.com/%s", utils.AWS_BUCKET_NAME, awsPath)

	// Update slide image with audio URL
	_, err = db.DB.Collection(collectionNameSlideImages).UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": bson.M{"audio_url": audioURL}})
	if err != nil {
		log.Println("Error updating slide image")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating slide image"})
		return
	}

	log.Println("Slide image updated with audio URL", audioURL)

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": audioURL, "status_code": http.StatusOK})
}

func generateFileName() string {
	return uuid.New().String()
}

// GenerateAudio2 is the updated version of GenerateAudio
func GenerateAudio2(c *gin.Context) {
	slideImageID := c.Param("slide_image_id")
	if slideImageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slide image ID is required"})
		return
	}

	update := c.Query("update")
	log.Println("UPDATE:", update)
	if update == "" {
		update = "false"
	}

	log.Println("*** /generate-audio-2 ***")
	log.Printf("Slide Image ID: %s", slideImageID)

	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(slideImageID)
	if err != nil {
		log.Println("Invalid slide image ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slide image ID"})
		return
	}

	var slideImage bson.M
	slideImageResult := db.DB.Collection(collectionNameSlideImages).FindOne(ctx, bson.M{"_id": objID})
	if err := slideImageResult.Decode(&slideImage); err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("Slide Image not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Slide Image not found"})
		} else {
			log.Println("Error finding slide image")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding slide image"})
		}
		return
	}

	generatedText, ok := slideImage["generated_text"].(string)
	if !ok {
		log.Println("Generated text not found")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Generated text not found"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	audioURL, ok := slideImage["audio_url"].(string)
	if ok && audioURL != "" && update == "false" {
		log.Println("Audio URL already exists")
		responseMap := map[string]interface{}{
			"status":      "success",
			"data":        audioURL,
			"status_code": http.StatusOK,
		}
		finalResponse, _ := json.Marshal(responseMap)
		fmt.Fprintf(c.Writer, "data: %s\n\n", string(finalResponse))
		c.Writer.Flush()
	}

	sendSSE := func(message string) {
		fmt.Fprintf(c.Writer, "data: %s\n\n", message)
		c.Writer.Flush()
	}

	sendSSE("Generating audio file")

	voice := "alloy"
	url := "https://api.openai.com/v1/audio/speech"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", utils.OPENAI_API_KEY),
	}
	data := map[string]interface{}{
		"model": "tts-1",
		"input": generatedText,
		"voice": voice,
		"speed": 1,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		sendSSE("Error preparing request data")
		return
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		sendSSE("Error creating request")
		return
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		sendSSE("Error sending request")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		sendSSE(fmt.Sprintf("Error generating audio file: %s", string(body)))
		return
	}
	sendSSE("Audio file generated")

	audioBlob, err := io.ReadAll(response.Body)
	if err != nil {
		sendSSE("Error reading audio response")
		return
	}
	sendSSE("Audio generated")

	fileName := generateFileName()
	awsPath := fmt.Sprintf("slides/%s/audio/%s.mp3", slideImage["slide_id"].(string), fileName)
	s3session := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(utils.AWS_REGION),
	}))
	uploader := s3.New(s3session)
	_, err = uploader.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(utils.AWS_BUCKET_NAME),
		Key:         aws.String(awsPath),
		Body:        bytes.NewReader(audioBlob),
		ContentType: aws.String("audio/mpeg"),
	})
	if err != nil {
		sendSSE("Error uploading audio to S3")
		return
	}
	sendSSE("Audio uploaded to S3")

	audioURL = fmt.Sprintf("https://%s.s3.amazonaws.com/%s", utils.AWS_BUCKET_NAME, awsPath)
	_, err = db.DB.Collection(collectionNameSlideImages).UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": bson.M{"audio_url": audioURL}})
	if err != nil {
		sendSSE("Error updating slide image")
		return
	}
	sendSSE("Slide image updated with audio URL")

	responseMap := map[string]interface{}{
		"status":      "success",
		"data":        audioURL,
		"status_code": http.StatusOK,
	}
	finalResponse, _ := json.Marshal(responseMap)
	fmt.Fprintf(c.Writer, "data: %s\n\n", string(finalResponse))
	c.Writer.Flush()
}

// GenerateAllAudioForSlide generates audio files for a given slide ID
func GenerateAllAudioForSlide(c *gin.Context) {
	// Get slide ID from request parameters
	slideID := c.Param("slide_id")

	// Find slide images by slide ID
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	_, err := primitive.ObjectIDFromHex(slideID)
	if err != nil {
		log.Println("Invalid slide ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slide ID"})
		return
	}

	slideImagesCursor, err := db.DB.Collection(collectionNameSlideImages).Find(ctx, bson.M{"slide_id": slideID})
	if err != nil {
		log.Println("Error finding slide images")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding slide images"})
		return
	}
	defer slideImagesCursor.Close(ctx)

	for slideImagesCursor.Next(ctx) {
		var slideImage bson.M
		err := slideImagesCursor.Decode(&slideImage)
		if err != nil {
			log.Println("Error decoding slide image")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding slide image"})
			return
		}

		order := slideImage["order"].(int32)
		log.Printf("Generating audio for slide image with order %d", order)

		// Check if audio URL already exists
		audioURL, ok := slideImage["audio_url"].(string)
		if ok && audioURL != "" {
			log.Println("Audio URL already exists")
			continue
		}

		// Generate audio for slide image
		err = generateAudioForSlideImage(ctx, slideImage)
		if err != nil {
			log.Println("Error generating audio for slide image")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating audio for slide image"})
			return
		}
	}

	log.Println("Audio generated for all slide images")
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Audio generated for all slide images"})
}

// GenerateAudioForSlideImage generates audio file for a given slide image
func generateAudioForSlideImage(ctx context.Context, slideImage bson.M) error {
	// Get generated text
	generatedText, ok := slideImage["generated_text"].(string)
	if !ok {
		log.Println("Generated text not found")
		return errors.New("generated text not found")
	}

	// Generate audio file
	log.Println("Generating audio file")
	voice := "alloy"
	url := "https://api.openai.com/v1/audio/speech"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", utils.OPENAI_API_KEY),
	}
	data := map[string]interface{}{
		"model": "tts-1",
		"input": generatedText,
		"voice": voice,
		"speed": 1,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println("Error preparing request data")
		return errors.New("error preparing request data")
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error creating request")
		return errors.New("error creating request")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request")
		return errors.New("error sending request")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		log.Println("Error generating audio file")
		return fmt.Errorf("error generating audio file: %s", string(body))
	}
	log.Println("Audio file generated")
	// Read audio content
	audioBlob, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println("Error reading audio response")
		return errors.New("error reading audio response")
	}
	log.Println("Audio generated")
	// Upload audio to S3
	fileName := generateFileName()
	awsPath := fmt.Sprintf("slides/%s/audio/%s.mp3", slideImage["slide_id"].(string), fileName)
	s3session := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(utils.AWS_REGION),
	}))
	uploader := s3.New(s3session)
	_, err = uploader.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(utils.AWS_BUCKET_NAME),
		Key:         aws.String(awsPath),
		Body:        bytes.NewReader(audioBlob),
		ContentType: aws.String("audio/mpeg"),
	})
	if err != nil {
		log.Println("Error uploading audio to S3")
		return errors.New("error uploading audio to S3")
	}
	log.Println("Audio uploaded to S3")
	// Generate audio URL
	audioURL := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", utils.AWS_BUCKET_NAME, awsPath)
	log.Println("Audio URL:", audioURL)
	// Update slide image with audio URL
	_, err = db.DB.Collection(collectionNameSlideImages).UpdateOne(ctx, bson.M{"_id": slideImage["_id"]}, bson.M{"$set": bson.M{"audio_url": audioURL}})
	if err != nil {
		log.Println("Error updating slide image", err)
		return errors.New("error updating slide image")
	}

	log.Println("Slide image updated with audio URL")
	return nil
}
