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

const CollectionNameSpaces = "spaces"

func CreateSpace(c *gin.Context) {
	log.Println("CreateSpace")

	var space models.Space

	if err := c.ShouldBindJSON(&space); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	space.ID = primitive.NewObjectID()
	space.CreatedAt = time.Now()
	space.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.DB.Collection(CollectionNameSpaces).InsertOne(ctx, space)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println("Create created:", result.InsertedID)

	c.JSON(http.StatusOK, result)
}

// Get all spaces
func GetSpaces(c *gin.Context) {
	log.Println("GetSpaces")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection(CollectionNameSpaces).Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var spaces []models.Space
	if err = cursor.All(ctx, &spaces); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, spaces)
}

// GetSpace
func GetSpace(c *gin.Context) {
	log.Println("GetSpace")

	spaceID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert spaceID to ObjectID
	objID, err := primitive.ObjectIDFromHex(spaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var space models.Space
	err = db.DB.Collection(CollectionNameSpaces).FindOne(ctx, bson.M{"_id": objID}).Decode(&space)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, space)
}

// GetSpaceSlides
func GetSpaceSlides(c *gin.Context) {
	log.Println("GetSpaceSlides")

	spaceID := c.Param("space_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection(CollectionNameSlides).Find(ctx, bson.M{"space_id": spaceID})
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

// DeleteSpace
func DeleteSpace(c *gin.Context) {
	log.Println("DeleteSpace")

	spaceID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert spaceID to ObjectID
	objID, err := primitive.ObjectIDFromHex(spaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := db.DB.Collection(CollectionNameSpaces).DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// // addSlideToSpace adds a slide to a space.
// func addSlideToSpace(c *gin.Context) {
// 	log.Println("addSlideToSpace")
// 	var slidespace models.SlideSpaceRequest
// 	if err := c.ShouldBindJSON(&slidespace); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	slide := db.DB.Collection(CollectionNameSlides).FindOne(context.TODO(), bson.M{"_id": bson.ObjectIdHex(slidespace.SlideID)})
// 	if slide.Err() != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"message": "Slide not found", "status_code": 404})
// 		return
// 	}
// 	result := db.DB.Collection(CollectionNameSlides).FindOneAndUpdate(context.TODO(), bson.M{"_id": bson.ObjectIdHex(slidespace.SlideID)}, bson.M{"$set": bson.M{"space_id": slidespace.SpaceID}}, options.FindOneAndUpdate().SetReturnDocument(options.After))
// 	if result.Err() != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Err().Error()})
// 		return
// 	}
// 	resultDoc := models.Slide{}
// 	if err := result.Decode(&resultDoc); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{"message": "Slide added to space successfully", "status_code": 200, "data": resultDoc})
// }

// // removeSlideFromSpace removes a slide from a space.
// func removeSlideFromSpace(c *gin.Context) {
// 	log.Println("removeSlideFromSpace")
// 	var slidespace models.SlideSpaceRequest
// 	if err := c.ShouldBindJSON(&slidespace); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	slide := db.DB.Collection(CollectionNameSlides).FindOne(context.TODO(), bson.M{"_id": bson.ObjectIdHex(slidespace.SlideID)})
// 	if slide.Err() != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"message": "Slide not found", "status_code": 404})
// 		return
// 	}
// 	result := db.DB.Collection(CollectionNameSlides).FindOneAndUpdate(context.TODO(), bson.M{"_id": bson.ObjectIdHex(slidespace.SlideID)}, bson.M{"$unset": bson.M{"space_id": ""}}, options.FindOneAndUpdate().SetReturnDocument(options.After))
// 	if result.Err() != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Err().Error()})
// 		return
// 	}
// 	resultDoc := models.Slide{}
// 	if err := result.Decode(&resultDoc); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{"message": "Slide removed from space successfully", "status_code": 200, "data": resultDoc})
// }
