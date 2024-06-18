package handlers

import (
	"log"
	"main/db"
	"main/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const CollectionNameAccessCodes = "access_codes"

func CreateAccessCode(c *gin.Context) {
	var accessCode models.AccessCode
	if err := c.ShouldBindJSON(&accessCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessCode.ID = primitive.NewObjectID()

	// Insert the access code into MongoDB
	_, err := db.DB.Collection(CollectionNameAccessCodes).InsertOne(c.Request.Context(), accessCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, accessCode)
}

// VerifyAccessCode verifies the access code and add to user
func VerifyAccessCode(c *gin.Context) {
	var accessCode models.AccessCode
	if err := c.ShouldBindJSON(&accessCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Println("Access code:", accessCode)
	// Find the access code by code in MongoDB
	var accessCodeFromDB models.AccessCode
	err := db.DB.Collection(CollectionNameAccessCodes).FindOne(c.Request.Context(), bson.M{"code": accessCode.Code}).Decode(&accessCodeFromDB)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Access code not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Update the access code in MongoDB and return the result
	_, err = db.DB.Collection(CollectionNameAccessCodes).UpdateOne(c.Request.Context(), bson.M{"code": accessCode.Code}, bson.M{"$set": bson.M{"used": true}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, accessCodeFromDB)
}

func GetAccessCode(c *gin.Context) {
	id := c.Param("id")

	// Find the access code by ID in MongoDB
	var accessCode models.AccessCode
	err := db.DB.Collection(CollectionNameAccessCodes).FindOne(c.Request.Context(), bson.M{"_id": id}).Decode(&accessCode)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Access code not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, accessCode)
}

func UpdateAccessCode(c *gin.Context) {
	id := c.Param("id")

	var accessCode models.AccessCode
	if err := c.ShouldBindJSON(&accessCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update the access code in MongoDB
	collection := db.DB.Collection(CollectionNameAccessCodes)
	_, err := collection.UpdateOne(c.Request.Context(), bson.M{"_id": id}, bson.M{"$set": accessCode})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, accessCode)
}

func DeleteAccessCode(c *gin.Context) {
	id := c.Param("id")

	// Delete the access code from MongoDB
	collection := db.DB.Collection(CollectionNameAccessCodes)
	_, err := collection.DeleteOne(c.Request.Context(), bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Access code deleted"})
}
