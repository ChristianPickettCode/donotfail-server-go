package handlers

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"main/db"
	"main/models"
	"main/utils"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gen2brain/go-fitz"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func ConvertPDFToImages(c *gin.Context) {
	slideID := c.Param("slide_id")
	fmt.Println("*** /convert-pdf-to-images ***")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(slideID)
	if err != nil {
		log.Println("Invalid slide ID", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid slide ID"})
		return
	}

	// Find slide
	var slide bson.M
	slideResult := db.DB.Collection("slides").FindOne(ctx, bson.M{"_id": objID})
	if err := slideResult.Decode(&slide); err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("Slide not found", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Slide not found"})
		} else {
			log.Println("Error finding slide", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding slide"})
		}
		return
	}

	pdfURL, ok := slide["pdf_url"].(string)
	if !ok || pdfURL == "" {
		log.Println("PDF URL not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "PDF URL not found"})
		return
	}

	log.Println("pdfURL", pdfURL)

	// Download the PDF
	response, err := http.Get(pdfURL)
	if err != nil {
		log.Println("Error downloading PDF", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error downloading PDF"})
		return
	}
	defer response.Body.Close()

	pdfBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println("Error reading PDF response", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading PDF response"})
		return
	}

	// Create a temporary directory for the PDF and images
	tmpDir, err := os.MkdirTemp(".", "fitz")
	if err != nil {
		log.Println("Error creating temp directory", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating temp directory"})
		return
	}
	defer os.RemoveAll(tmpDir)

	// Save the PDF to the temporary directory
	tempPDFPath := filepath.Join(tmpDir, "document.pdf")
	if err := ioutil.WriteFile(tempPDFPath, pdfBytes, 0644); err != nil {
		log.Println("Error writing PDF to file", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error writing PDF to file"})
		return
	}

	// Convert PDF to images
	images, err := pdfToImages(tempPDFPath)
	if err != nil {
		log.Println("Error converting PDF to images", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error converting PDF to images"})
		return
	}

	// Save images to the temporary directory and upload them to S3
	log.Println("Number of images: ", len(images))
	index := 0
	for _, img := range images {
		fileName := generateFileName()
		imagePath := filepath.Join(tmpDir, fileName)
		err = saveImageToFile(img, imagePath)
		if err != nil {
			log.Println("Error saving image to file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving image to file"})
			return
		}
		err = uploadFileToS3(c, slideID, imagePath, fileName, "image/png", index)
		if err != nil {
			log.Println("Error uploading image to S3", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error uploading image to S3"})
			return
		}
		index++
	}

	c.JSON(http.StatusOK, gin.H{"message": "PDF converted to images", "status_code": 200})
}

func pdfToImages(pdfPath string) ([]image.Image, error) {
	doc, err := fitz.New(pdfPath)
	if err != nil {
		return nil, err
	}
	defer doc.Close()

	var images []image.Image
	for n := 0; n < doc.NumPage(); n++ {
		img, err := doc.Image(n)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, nil
}

func saveImageToFile(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	err = png.Encode(file, img)
	if err != nil {
		return err
	}

	return nil
}

func uploadFileToS3(c *gin.Context, slideID string, filePath string, fileName string, contentType string, index int) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	awsPath := fmt.Sprintf("slides/%s/%s", slideID, fileName)

	s3session := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(utils.AWS_REGION),
	}))
	uploader := s3.New(s3session)
	_, err = uploader.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(utils.AWS_BUCKET_NAME),
		Key:         aws.String(awsPath),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", utils.AWS_BUCKET_NAME, awsPath)
	log.Println("Uploaded file URL:", url)

	if contentType == "image/png" {
		slideImage := models.SlideImage{
			ID:        primitive.NewObjectID(),
			SlideID:   slideID,
			ImageURL:  url,
			Order:     index,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		result, err := db.DB.Collection(collectionNameSlideImages).InsertOne(context.TODO(), slideImage)
		if err != nil {
			return err
		}
		log.Println("Inserted slide image with ID:", result.InsertedID)
	}

	return nil
}
