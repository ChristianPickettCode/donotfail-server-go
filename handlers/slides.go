package handlers

import (
	"context"
	"log"
	"main/db"
	"main/models"
	"main/utils"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const CollectionNameSlides = "slides"

// Get all slides
func GetSlides(c *gin.Context) {
	log.Println("getSlides")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection(CollectionNameSlides).Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var slides []models.Slide
	if err = cursor.All(ctx, &slides); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, slides)
}

// GetSlide
func GetSlide(c *gin.Context) {
	log.Println("GetSlide")

	slideID := c.Param("id")

	objID, err := primitive.ObjectIDFromHex(slideID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var slide models.Slide
	err = db.DB.Collection(CollectionNameSlides).FindOne(ctx, bson.M{"_id": objID}).Decode(&slide)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Slide not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, slide)
}

// CreateSlide
func CreateSlide(c *gin.Context) {
	log.Println("CreateSlide")

	var slide models.Slide

	if err := c.ShouldBindJSON(&slide); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slide.ID = primitive.NewObjectID()
	slide.CreatedAt = time.Now()
	slide.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.DB.Collection(CollectionNameSlides).InsertOne(ctx, slide)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Create created:", result.InsertedID)

	c.JSON(http.StatusOK, result)
}

// UpdateSlide
func UpdateSlide(c *gin.Context) {
	log.Println("UpdateSlide")

	// Get slide id
	slideID := c.Param("id")

	// Convert slideID to ObjectID
	objID, err := primitive.ObjectIDFromHex(slideID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var slide models.Slide

	if err := c.ShouldBindJSON(&slide); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slide.UpdatedAt = time.Now()

	log.Println("Slide:", slide)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{}
	slideType := reflect.TypeOf(slide)
	slideValue := reflect.ValueOf(slide)
	for i := 0; i < slideType.NumField(); i++ {
		field := slideType.Field(i)
		fieldValue := slideValue.Field(i).Interface()
		fieldType := field.Type.Kind()

		if fieldType == reflect.Bool || !reflect.DeepEqual(fieldValue, reflect.Zero(field.Type).Interface()) {
			bsonTag := field.Tag.Get("bson")
			// Skip if bson tag is not set or is "-"
			if bsonTag == "" || bsonTag == "-" {
				continue
			}

			update[field.Tag.Get("bson")] = fieldValue
		}
	}

	result, err := db.DB.Collection(CollectionNameSlides).UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": update})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Update result:", result)

	c.JSON(http.StatusOK, result)
}

// deleteAWSFile deletes a file from AWS S3 given its key
func deleteAWSFile(key string) bool {
	log.Println("Deleting file from S3:", key)
	s3session := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	}))
	svc := s3.New(s3session)
	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(utils.AWS_BUCKET_NAME),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Println("Error deleting file from S3:", err)
		return false
	}

	log.Println("File deletion initiated for key:", key)
	return true
}

// DeleteSlide is a gin handler to delete a slide
// Add more logs
func DeleteSlide(c *gin.Context) {
	slideID := c.Param("id")
	log.Println("*** /delete_slide ***")
	log.Println("Slide ID:", slideID)

	objID, err := primitive.ObjectIDFromHex(slideID)
	if err != nil {
		log.Println("Invalid slide ID", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slide ID"})
		return
	}

	// Find the slide
	var slide bson.M
	err = db.DB.Collection("slides").FindOne(context.Background(), bson.M{"_id": objID}).Decode(&slide)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("Slide not found", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Slide not found"})
		} else {
			log.Println("Error finding slide", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding slide"})
		}
		return
	}

	log.Print("Found slide:", slide)

	pdfURL, _ := slide["pdf_url"].(string)

	// Delete the slide
	result, err := db.DB.Collection(CollectionNameSlides).DeleteOne(context.Background(), bson.M{"_id": objID})
	if err != nil {
		log.Println("Error deleting slide", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting slide"})
		return
	}

	log.Println("Delete result:", result)

	if result.DeletedCount == 1 {
		if pdfURL != "" {
			pdfKey := strings.Split(pdfURL, "amazonaws.com/")[1]
			deleteAWSFile(pdfKey)
		}

		// Delete all images and audio files associated with the slide
		cursor, err := db.DB.Collection(collectionNameSlideImages).Find(context.Background(), bson.M{"slide_id": slideID})
		if err != nil {
			log.Println("Error finding slide images", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding slide images"})
			return
		}
		defer cursor.Close(context.Background())

		for cursor.Next(context.Background()) {
			log.Println("Deleting slide image and audio file")
			var slideImage bson.M
			if err := cursor.Decode(&slideImage); err != nil {
				log.Println("Error decoding slide image", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding slide image"})
				return
			}
			log.Println("Slide image:", slideImage["_id"])

			audioURL, _ := slideImage["audio_url"].(string)
			if audioURL != "" {
				log.Println("Deleting audio file")
				audioKey := strings.Split(audioURL, "amazonaws.com/")[1]
				deleteAWSFile(audioKey)
			}

			log.Println("Deleting slide image document")
			imageURL := slideImage["image_url"].(string)
			imageKey := strings.Split(imageURL, "amazonaws.com/")[1]
			deleteAWSFile(imageKey)
		}

		log.Println("Deleting slide images documents")

		_, err = db.DB.Collection(collectionNameSlideImages).DeleteMany(context.Background(), bson.M{"slide_id": slideID})
		if err != nil {
			log.Println("Error deleting slide images", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting slide images"})
			return
		}

		log.Println("Slide deleted successfully")

		c.JSON(http.StatusOK, gin.H{"message": "Slide deleted successfully", "status_code": 200})
	} else {

		log.Println("Slide not found")
		c.JSON(http.StatusNotFound, gin.H{"message": "Slide not found", "status_code": 404})
	}
}
