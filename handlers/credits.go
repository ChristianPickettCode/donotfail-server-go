package handlers

import (
	"context"
	"log"
	"main/db"
	"main/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

// const credit values
const (
	GENERATENOTESFORSLIDE = 1
	GENERATEQUIZQUESTION  = 1
	GENERATEAUDIO         = 2
)

func AddCredits(c *gin.Context) {
	log.Println("AddCredits")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userID := c.Param("user_id")
	amount := c.Param("amount")

	// Parse amount as an integer
	credits, err := strconv.Atoi(amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount"})
		return
	}

	var user models.User

	if err := db.DB.Collection(CollectionNameUsers).FindOne(ctx, bson.M{"user_id": userID}).Decode(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add credits to the user
	user.Credits += credits

	// Update the user in MongoDB
	_, err = db.DB.Collection(CollectionNameUsers).UpdateOne(ctx, bson.M{"user_id": userID}, bson.M{"$set": bson.M{"credits": user.Credits}})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Credits added successfully", "credits": user.Credits, "status_code": http.StatusOK})
}

func RemoveCredits(c *gin.Context) {
	log.Println("RemoveCredits")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userID := c.Param("user_id")
	amount := c.Param("amount")

	// Parse amount as an integer
	credits, err := strconv.Atoi(amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount"})
		return
	}

	var user models.User

	if err := db.DB.Collection(CollectionNameUsers).FindOne(ctx, bson.M{"user_id": userID}).Decode(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("User credits:", user.Credits)
	// Check if user has enough credits
	if user.Credits < credits {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient credits", "status_code": http.StatusBadRequest, "credits": user.Credits})
		return
	}

	// Remove credits from the user
	user.Credits -= credits

	// Update the user in MongoDB
	_, err = db.DB.Collection(CollectionNameUsers).UpdateOne(ctx, bson.M{"user_id": userID}, bson.M{"$set": bson.M{"credits": user.Credits}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Credits removed successfully", "credits": user.Credits, "status_code": http.StatusOK})
}

func GetUserCredits(c *gin.Context) {
	log.Println("GetCredits")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userID := c.Param("user_id")

	var user models.User

	if err := db.DB.Collection(CollectionNameUsers).FindOne(ctx, bson.M{"user_id": userID}).Decode(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"credits": user.Credits, "status_code": http.StatusOK})
}
