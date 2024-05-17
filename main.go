package main

import (
	"log"
	"main/db"
	"main/handlers"
	"main/utils"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("Starting server")

	r := gin.Default()
	r.Use(cors.Default())

	log.Println("Loading .env file")
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	utils.LoadEnvs()

	// stripeSecret := os.Getenv("STRIPE_SECRET")

	// if stripeSecret == "" {
	// 	log.Fatal("STRIPE_SECRET not found in .env file")
	// }
	// stripe.Key = stripeSecret

	log.Println("Connecting to MongoDB")
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI not found in .env file")
	}
	db.ConnectMongo(mongoURI)

	log.Println("Setting up routes")
	handlers.SetUpRoutes(r)

	// Start the server
	r.Run(":8080") // listen and serve on 0.0.0.0:8080
}
