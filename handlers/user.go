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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const CollectionNameUsers = "users"

func CreateUser(c *gin.Context) {
	log.Println("CreateUser")

	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.SpaceIDs = []string{}
	user.Credits = 100

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.DB.Collection(CollectionNameUsers).InsertOne(ctx, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Create created:", result.InsertedID)

	c.JSON(http.StatusOK, result)
}

// Get user
func GetUser(c *gin.Context) {
	log.Println("GetUser")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	userID := c.Param("user_id")

	if err := db.DB.Collection(CollectionNameUsers).FindOne(ctx, bson.M{"user_id": userID}).Decode(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// Get user spaces
func GetUserSpaces(c *gin.Context) {
	log.Println("GetUserSpaces")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	userID := c.Param("user_id")

	if err := db.DB.Collection(CollectionNameUsers).FindOne(ctx, bson.M{"user_id": userID}).Decode(&user); err != nil {
		log.Println("Error finding user:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var userSpaces []models.Space
	for _, spaceID := range user.SpaceIDs {
		log.Println("Space ID:", spaceID)
		var space models.Space

		objID, err := primitive.ObjectIDFromHex(spaceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := db.DB.Collection(CollectionNameSpaces).FindOne(ctx, bson.M{"_id": objID}).Decode(&space); err != nil {
			log.Println("Error finding space:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		userSpaces = append(userSpaces, space)
	}

	log.Println("User spaces:", userSpaces)

	c.JSON(http.StatusOK, userSpaces)
}

// Add space to user from user ID and space ID
func AddSpaceToUser(c *gin.Context) {
	log.Println("AddSpaceToUser")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	userID := c.Param("user_id")
	spaceID := c.Param("space_id")

	if err := db.DB.Collection(CollectionNameUsers).FindOne(ctx, bson.M{"user_id": userID}).Decode(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if the spaceID is already present in user.SpaceIDs
	alreadyPresent := false
	for _, id := range user.SpaceIDs {
		if id == spaceID {
			alreadyPresent = true
			break
		}
	}

	// Only add the spaceID if it's not already present
	if !alreadyPresent {
		user.SpaceIDs = append(user.SpaceIDs, spaceID)

		update := bson.M{
			"$set": bson.M{
				"space_ids": user.SpaceIDs,
			},
		}

		if _, err := db.DB.Collection(CollectionNameUsers).UpdateOne(ctx, bson.M{"user_id": userID}, update); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// If the spaceID is already present, you might want to notify the client in some way
		c.JSON(http.StatusOK, gin.H{"message": "Space already added to user"})
		return
	}
}

// Remove space from user from user ID and space ID
func RemoveSpaceFromUser(c *gin.Context) {
	log.Println("RemoveSpaceFromUser")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	userID := c.Param("user_id")
	spaceID := c.Param("space_id")

	if err := db.DB.Collection(CollectionNameUsers).FindOne(ctx, bson.M{"user_id": userID}).Decode(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var newSpaceIDs []string
	for _, id := range user.SpaceIDs {
		if id != spaceID {
			newSpaceIDs = append(newSpaceIDs, id)
		}
	}

	update := bson.M{
		"$set": bson.M{
			"space_ids": newSpaceIDs,
		},
	}

	if _, err := db.DB.Collection(CollectionNameUsers).UpdateOne(ctx, bson.M{"user_id": userID}, update); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}
