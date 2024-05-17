package handlers

import (
	"context"
	"log"
	"main/db"
	"main/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

const CollectionNameSlideImages = "slide_images"

// get all slide images
func GetSlideImages(c *gin.Context) {
	log.Println("getSlideImages")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	slideID := c.Param("slide_id")
	cursor, err := db.DB.Collection(CollectionNameSlideImages).Find(ctx, bson.M{"slide_id": slideID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var slideImages []models.SlideImage
	if err = cursor.All(ctx, &slideImages); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, slideImages)
}
